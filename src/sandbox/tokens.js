import { randomBytes } from 'crypto';

/**
 * Generate a cryptographically random, URL-safe agent token.
 * Uses base64url encoding (no padding) — produces a 43-character string.
 * Example: aB6Ye_fia5UH8k1KGTbpoPzDtYfaPqyiTcfsxR6X9EE
 */
export function generateAgentToken() {
  return randomBytes(32).toString('base64url');
}
