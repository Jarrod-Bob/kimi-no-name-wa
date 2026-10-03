import { describe, expect, it } from 'vitest';
import { parseError } from './api';

describe('parseError', () => {
  it('reads the API error envelope', () => {
    const err = parseError(503, '{"error":{"message":"Ollama isn\'t reachable.","code":"ollama_unreachable"}}');
    expect(err.message).toBe("Ollama isn't reachable.");
    expect(err.code).toBe('ollama_unreachable');
    expect(err.status).toBe(503);
  });

  it('falls back for anything else', () => {
    expect(parseError(500, '<html>').message).toBe('The server answered 500.');
  });
});
