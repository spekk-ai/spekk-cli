import { mkdirSync, existsSync } from 'node:fs';
import { join } from 'node:path';

export async function showSpekk() {
  // Create .spekk directory if it doesn't exist
  const spekkDir = join(process.cwd(), '.spekk');
  
  if (!existsSync(spekkDir)) {
    mkdirSync(spekkDir, { recursive: true });
    console.log('Created .spekk directory');
  }
  
  // For now, just output a basic message
  // Future iterations will generate HTML and open browser
  console.log('Spekk show command executed successfully');
  console.log('Directory:', process.cwd());
  console.log('.spekk directory:', existsSync(spekkDir) ? 'exists' : 'missing');
}