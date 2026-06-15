package show

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/parser"
)

// sseClient script injected before </body> in watch mode.
// Saves UI state (expanded specs, active detail, scroll position) to
// sessionStorage before reloading, and restores it after reload.
const sseClientScript = `
<script>
(function() {
  function restoreWatchState() {
    var raw = sessionStorage.getItem('spekkWatchState');
    if (!raw) return;
    sessionStorage.removeItem('spekkWatchState');
    try { var state = JSON.parse(raw); } catch(e) { return; }

    if (state.expandedSpecs && state.expandedSpecs.length) {
      state.expandedSpecs.forEach(function(specId) {
        var assertions = document.getElementById('assertions-' + specId);
        var toggle = document.getElementById('toggle-' + specId);
        if (assertions && toggle) {
          assertions.classList.add('expanded');
          toggle.classList.add('expanded');
          toggle.parentElement.classList.add('expanded');
          toggle.textContent = '\u25BC';
        }
      });
    }

    if (state.activeDetailId) {
      var detailEl = document.getElementById(state.activeDetailId);
      if (detailEl) {
        document.querySelectorAll('.detail-content').forEach(function(el) {
          el.classList.remove('active');
        });
        detailEl.classList.add('active');
        var emptyState = document.getElementById('empty-state');
        if (emptyState) emptyState.style.display = 'none';
        var match = state.activeDetailId.match(/^detail-(assertion|spec)-(.+)$/);
        if (match) {
          if (match[1] === 'assertion') {
            var item = document.querySelector('.assertion-item[data-assertion-id="' + match[2] + '"]');
            if (item) item.classList.add('selected');
          } else {
            var toggleEl = document.getElementById('toggle-' + match[2]);
            if (toggleEl) toggleEl.parentElement.classList.add('selected');
          }
        }
      }
    }

    if (typeof state.scrollTop === 'number') {
      var treePanel = document.querySelector('.tree-panel');
      if (treePanel) requestAnimationFrame(function() { treePanel.scrollTop = state.scrollTop; });
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', restoreWatchState);
  } else {
    restoreWatchState();
  }

  var es = new EventSource('/events');
  es.addEventListener('reload', function() {
    var expandedSpecs = [];
    document.querySelectorAll('[id^="assertions-"].expanded').forEach(function(el) {
      expandedSpecs.push(el.id.replace('assertions-', ''));
    });
    var activeDetail = document.querySelector('.detail-content.active');
    var activeDetailId = activeDetail ? activeDetail.id : null;
    var treePanel = document.querySelector('.tree-panel');
    var scrollTop = treePanel ? treePanel.scrollTop : 0;
    sessionStorage.setItem('spekkWatchState', JSON.stringify({
      expandedSpecs: expandedSpecs,
      activeDetailId: activeDetailId,
      scrollTop: scrollTop
    }));
    location.reload();
  });
})();
</script>`

// RunWatch starts the watch-mode HTTP server with SSE live reload.
//
// opts mirrors show.Run's Options so the watch path stays consistent. The
// cross-branch fields are threaded through but not yet acted upon; watch
// behavior is unchanged until later work fills in the real cross-branch flow.
func RunWatch(specsDir string, opts Options) error {
	if opts.CrossBranch {
		// Placeholder: cross-branch diffing is implemented by later waves.
		fmt.Fprintln(os.Stderr, "cross-branch mode requested (not yet implemented)")
	}

	var (
		mu    sync.Mutex
		dirty bool
	)

	// getHTML regenerates HTML from current spec state.
	getHTML := func() (string, error) {
		mu.Lock()
		dirty = false
		mu.Unlock()

		result, err := parser.ParseAllSpecs(specsDir)
		if err != nil {
			return "", fmt.Errorf("parsing specs: %w", err)
		}
		if len(result.Specs) == 0 {
			return "", fmt.Errorf("no specifications found in %s", specsDir)
		}

		data := buildShowData(specsDir, result)
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return "", fmt.Errorf("marshaling data: %w", err)
		}

		html := strings.Replace(templateHTML, "/*__SPEKK_DATA__*/", string(jsonBytes), 1)
		// Inject SSE client script before </body>
		html = strings.Replace(html, "</body>", sseClientScript+"\n</body>", 1)
		return html, nil
	}

	// SSE client management
	type sseClient chan struct{}
	var (
		clientsMu sync.Mutex
		clients   = make(map[sseClient]struct{})
	)

	addClient := func() sseClient {
		ch := make(sseClient, 1)
		clientsMu.Lock()
		clients[ch] = struct{}{}
		clientsMu.Unlock()
		return ch
	}

	removeClient := func(ch sseClient) {
		clientsMu.Lock()
		delete(clients, ch)
		clientsMu.Unlock()
	}

	notifyClients := func() {
		clientsMu.Lock()
		defer clientsMu.Unlock()
		for ch := range clients {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}

	// HTTP handler
	mux := http.NewServeMux()

	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "\n")
		flusher.Flush()

		ch := addClient()
		defer removeClient(ch)

		// If changes happened before SSE connected, send immediate reload
		mu.Lock()
		wasDirty := dirty
		mu.Unlock()
		if wasDirty {
			fmt.Fprint(w, "event: reload\ndata: reload\n\n")
			flusher.Flush()
		}

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ch:
				fmt.Fprint(w, "event: reload\ndata: reload\n\n")
				flusher.Flush()
			}
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "" {
			http.NotFound(w, r)
			return
		}
		html, err := getHTML()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	// Find available port starting at 3117
	listener, err := listenWithRetry(3117, 10)
	if err != nil {
		return fmt.Errorf("binding to port: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	server := &http.Server{Handler: mux}

	// Start file watcher
	stopWatcher := watchSpecs(specsDir, func() {
		mu.Lock()
		dirty = true
		mu.Unlock()
		notifyClients()
	})

	// Handle SIGINT for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve(listener)
	}()

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	fmt.Fprintf(os.Stderr, "Spec Explorer watching at %s (press Ctrl+C to stop)\n", url)

	// Open in browser (skip in CI)
	if os.Getenv("CI") == "" {
		openBrowser(url)
	}

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "\nShutting down...")
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			stopWatcher()
			return fmt.Errorf("server error: %w", err)
		}
	}

	stopWatcher()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)

	return nil
}

// listenWithRetry tries to listen on 127.0.0.1:port, incrementing the port
// up to maxRetries times if the port is in use.
func listenWithRetry(port, maxRetries int) (net.Listener, error) {
	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		addr := fmt.Sprintf("127.0.0.1:%d", port+i)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			return ln, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// watchSpecs polls specsDir for .md file changes every 500ms.
// Calls onChange when files are added, modified, or deleted.
// Returns a stop function.
func watchSpecs(specsDir string, onChange func()) func() {
	snapshot := scanMdFiles(specsDir)

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				current := scanMdFiles(specsDir)
				if snapshotChanged(snapshot, current) {
					snapshot = current
					onChange()
				}
			}
		}
	}()

	return func() { close(done) }
}

// scanMdFiles recursively scans dir for .md files and returns a map of
// relative path to modification time.
func scanMdFiles(dir string) map[string]time.Time {
	result := make(map[string]time.Time)
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		result[rel] = info.ModTime()
		return nil
	})
	return result
}

// snapshotChanged compares two file snapshots for any differences.
func snapshotChanged(prev, curr map[string]time.Time) bool {
	if len(prev) != len(curr) {
		return true
	}
	for path, mtime := range prev {
		currMtime, ok := curr[path]
		if !ok || !currMtime.Equal(mtime) {
			return true
		}
	}
	return false
}
