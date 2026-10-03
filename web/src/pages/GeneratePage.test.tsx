// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { Generation, GenerateRequest } from '../api';
import { AppStateProvider } from '../session/AppState';
import { GeneratePage } from './GeneratePage';

const health = {
  status: 'ok',
  ollama: { url: 'http://127.0.0.1:11434', reachable: true, model: 'gemma4:31b', model_pulled: true, models: ['gemma4:31b'] },
};

function generation(id: number, req: GenerateRequest, names: string[]): Generation {
  return {
    id,
    created_at: '2026-10-03T00:00:00Z',
    client: 'web',
    model: 'gemma4:31b',
    description: req.description,
    inspiration: req.inspiration ?? [],
    keywords: req.keywords ?? [],
    tones: req.tones ?? [],
    like_name: req.like?.name ?? '',
    names: names.map((name, i) => ({
      id: id * 100 + i,
      generation_id: id,
      name,
      technique: 'portmanteau',
      explanation: `${name} explained`,
      tone: 'witty',
      favourite: false,
      favourited_at: null,
    })),
  };
}

let requests: GenerateRequest[];
let nextNames: string[][];

beforeEach(() => {
  sessionStorage.clear();
  requests = [];
  nextNames = [
    ['acceleread', 'skimurai'],
    ['pagewright', 'readrunner'],
    ['inkling', 'yomi'],
  ];
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string, init?: RequestInit) => {
      if (url === '/api/v1/health') return new Response(JSON.stringify(health));
      if (url === '/api/v1/names') {
        const req = JSON.parse(String(init?.body)) as GenerateRequest;
        requests.push(req);
        return new Response(JSON.stringify(generation(requests.length, req, nextNames.shift() ?? [])));
      }
      if (url.endsWith('/favourite')) {
        return new Response(
          JSON.stringify({ id: 100, generation_id: 1, name: 'acceleread', technique: 'portmanteau', explanation: 'x', tone: 'witty', favourite: init?.method === 'PUT', favourited_at: null }),
        );
      }
      return new Response('{"error":{"message":"nope","code":"not_found"}}', { status: 404 });
    }),
  );
  Element.prototype.scrollIntoView = vi.fn();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function renderPage() {
  return render(
    <MemoryRouter>
      <AppStateProvider>
        <GeneratePage />
      </AppStateProvider>
    </MemoryRouter>,
  );
}

async function settle() {
  await act(async () => {
    await new Promise((r) => setTimeout(r, 0));
  });
}

describe('GeneratePage', () => {
  it('generates from the form, then re-rolls avoiding every shown name', async () => {
    renderPage();
    fireEvent.change(screen.getByLabelText('Describe it'), { target: { value: 'A tool to ingest documents quicker' } });
    const inspiration = screen.getByLabelText('Surroundings & inspiration');
    fireEvent.change(inspiration, { target: { value: 'maplestory' } });
    fireEvent.keyDown(inspiration, { key: 'Enter' });
    fireEvent.click(screen.getByRole('button', { name: /Japanese-flavoured/ }));

    fireEvent.click(screen.getByRole('button', { name: 'Name it' }));
    await settle();

    expect(requests[0]).toMatchObject({
      description: 'A tool to ingest documents quicker',
      inspiration: ['maplestory'],
      tones: ['witty', 'punny', 'sleek', 'japanese'],
      count: 8,
      avoid: [],
      client: 'web',
    });
    expect(screen.getByRole('heading', { name: 'acceleread' })).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: /Re-roll/ }));
    await settle();
    expect(requests[1].avoid).toEqual(['acceleread', 'skimurai']);

    fireEvent.click(screen.getByRole('button', { name: /Re-roll/ }));
    await settle();
    expect(requests[2].avoid).toEqual(['acceleread', 'skimurai', 'pagewright', 'readrunner']);
  });

  it('asks for more like a name, carrying the seed', async () => {
    renderPage();
    fireEvent.change(screen.getByLabelText('Describe it'), { target: { value: 'fast reader' } });
    fireEvent.keyDown(screen.getByLabelText('Describe it'), { key: 'Enter', ctrlKey: true });
    await settle();

    const card = screen.getByRole('article', { name: 'skimurai' });
    fireEvent.click(within(card).getByRole('button', { name: /More like this/ }));
    await settle();

    expect(requests[1].like).toEqual({ name: 'skimurai', technique: 'portmanteau', explanation: 'skimurai explained' });
    expect(requests[1].avoid).toEqual(['acceleread', 'skimurai']);
    expect(screen.getByRole('heading', { name: /More like “skimurai”/ })).toBeTruthy();
  });

  it('stars a name and shows it pressed', async () => {
    renderPage();
    fireEvent.change(screen.getByLabelText('Describe it'), { target: { value: 'fast reader' } });
    fireEvent.click(screen.getByRole('button', { name: 'Name it' }));
    await settle();

    const star = screen.getByRole('button', { name: 'Star acceleread' });
    expect(star.getAttribute('aria-pressed')).toBe('false');
    fireEvent.click(star);
    await settle();
    expect(screen.getByRole('button', { name: 'Unstar acceleread' }).getAttribute('aria-pressed')).toBe('true');
  });

  it('refuses to generate without a description', async () => {
    renderPage();
    fireEvent.submit(screen.getByRole('form', { name: /What are you naming/ }));
    await settle();
    expect(requests).toHaveLength(0);
    expect(screen.getByRole('alert').textContent).toMatch(/Describe what you’re naming/);
  });
});
