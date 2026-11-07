import { randomUUID } from 'node:crypto';
import { pathToFileURL } from 'node:url';
import { createServer } from 'vite';
import react from '@vitejs/plugin-react';

/**
 * Isolated Arena sandbox preview server for exploring the real React UI when
 * the external Go/MSSQL service is not running locally. Production and
 * vite.config.js remain completely untouched.
 */
function createSyntheticPreviewApi() {
  const usersByPhone = new Map();
  const sessions = new Map();

  // Pre-seed 09000000002 as a completed returning profile so reviewers can also
  // jump straight to Home, Drawer, Letters, and Chat if desired, while any other
  // 09xxxxxxxxx number (such as 09000000001) starts at Welcome for full onboarding.
  const seededConvId = '11111111-1111-4111-8111-111111111111';
  usersByPhone.set('09000000002', {
    userId: '22222222-2222-4222-8222-222222222222',
    displayName: 'مهدی',
    conversationStyle: 'balanced',
    onboardingCompletedAt: '2026-11-07T08:00:00.000Z',
    furthestStep: 25,
    privacySharing: true,
    conversations: [
      {
        id: seededConvId,
        title: 'نخستین پرتو صبحگاهی',
        createdAt: '2026-11-07T08:05:00.000Z',
        updatedAt: '2026-11-07T08:10:00.000Z',
        messages: [
          {
            id: '33333333-3333-4333-8333-333333333333',
            role: 'user',
            content: 'این روزها ذهنم پر از کارهای ناتمامه و دنبال کمی آرامش و شفافیت می‌گردم.',
            createdAt: '2026-11-07T08:05:00.000Z',
          },
          {
            id: '44444444-4444-4444-8444-444444444444',
            role: 'assistant',
            content: 'خوبه که اینجا ازش گفتی.\n\nوقتی همه‌چیز هم‌زمان توی ذهن جمع می‌شه، طبیعی‌ست که دیدنت تار بشه. بیا مثل نور اول صبح، فقط روی یک نقطهٔ کوچک بایستیم: الان کدوم موضوع بیشترین سنگینی رو داره؟',
            createdAt: '2026-11-07T08:06:00.000Z',
          },
        ],
      },
    ],
    bookmarks: new Map(),
    letters: [
      {
        id: '55555555-5555-4555-8555-555555555555',
        kind: 'personalized',
        title: 'روشناییِ مکث پیش از تصمیم',
        salutation: 'دوست من،',
        paragraphs: [
          'در گفت‌وگوی امروزمان دیدیم که وقتی همهٔ کارها را یک‌جا نگاه می‌کنی، ذهن پیش از حرکت خسته می‌شود.',
          'سپیده‌دم هم جهان را یک‌باره روشن نمی‌کند؛ آرام‌آرام از یک خط باریک در افق آغاز می‌شود و به چشم فرصت می‌دهد دوباره ببیند.',
        ],
        question: 'امروز کدام قدم کوچک برایت حکم همان نخستین پرتو روشن را دارد؟',
        createdAt: '2026-11-07T09:00:00.000Z',
        sourceConversationIds: [seededConvId],
      },
    ],
  });

  const getOrCreateUser = (phone) => {
    if (!usersByPhone.has(phone)) {
      usersByPhone.set(phone, {
        userId: randomUUID(),
        displayName: '',
        conversationStyle: 'balanced',
        onboardingCompletedAt: null,
        furthestStep: 0,
        privacySharing: true,
        conversations: [],
        bookmarks: new Map(),
        letters: [],
      });
    }
    return usersByPhone.get(phone);
  };

  const parseCookies = (cookieHeader = '') => Object.fromEntries(
    cookieHeader.split(';').map((part) => part.trim()).filter(Boolean).map((pair) => {
      const idx = pair.indexOf('=');
      return idx < 0 ? [pair, ''] : [pair.slice(0, idx), decodeURIComponent(pair.slice(idx + 1))];
    }),
  );

  const readJson = (req) => new Promise((resolve) => {
    let raw = '';
    req.on('data', (chunk) => { raw += chunk; });
    req.on('end', () => {
      try { resolve(raw ? JSON.parse(raw) : {}); }
      catch { resolve({}); }
    });
  });

  const sendJson = (res, status, payload, extraHeaders = {}) => {
    res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8', ...extraHeaders });
    res.end(JSON.stringify(payload));
  };

  return async (req, res, next) => {
    const url = new URL(req.url || '/', 'http://127.0.0.1');
    if (!url.pathname.startsWith('/api/v1/')) return next();

    const cookies = parseCookies(req.headers.cookie);
    const sid = cookies.theia_preview_sid;
    const user = sid ? sessions.get(sid) : null;

    if (req.method === 'POST' && url.pathname === '/api/v1/dev/identity-session') {
      const body = await readJson(req);
      const phone = String(body.phone || '').trim();
      if (!/^09\d{9}$/.test(phone)) {
        return sendJson(res, 400, { code: 'INVALID_PHONE', title: 'Invalid phone' });
      }
      const account = getOrCreateUser(phone);
      const token = randomUUID();
      sessions.set(token, account);
      return sendJson(
        res,
        200,
        {
          userId: account.userId,
          displayName: account.displayName,
          conversationStyle: account.conversationStyle,
          onboardingCompletedAt: account.onboardingCompletedAt,
        },
        { 'Set-Cookie': `theia_preview_sid=${token}; Path=/; HttpOnly; SameSite=Lax` },
      );
    }

    if (req.method === 'DELETE' && url.pathname === '/api/v1/dev/session') {
      if (sid) sessions.delete(sid);
      res.writeHead(204, { 'Set-Cookie': 'theia_preview_sid=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax' });
      res.end();
      return undefined;
    }

    if (!user) {
      return sendJson(res, 401, { code: 'UNAUTHORIZED', title: 'Not signed in' });
    }

    if (req.method === 'GET' && url.pathname === '/api/v1/me') {
      return sendJson(res, 200, {
        userId: user.userId,
        displayName: user.displayName,
        conversationStyle: user.conversationStyle,
        onboardingCompletedAt: user.onboardingCompletedAt,
      });
    }

    if (req.method === 'PATCH' && url.pathname === '/api/v1/me/onboarding-name') {
      const body = await readJson(req);
      user.displayName = String(body.displayName || '').trim().slice(0, 60);
      return sendJson(res, 200, {
        userId: user.userId,
        displayName: user.displayName,
        conversationStyle: user.conversationStyle,
        onboardingCompletedAt: null,
      });
    }

    if (req.method === 'PATCH' && url.pathname === '/api/v1/me/profile') {
      const body = await readJson(req);
      if (typeof body.displayName === 'string' && body.displayName.trim()) {
        user.displayName = body.displayName.trim().slice(0, 60);
      }
      if (typeof body.conversationStyle === 'string') {
        user.conversationStyle = body.conversationStyle;
      }
      if (!user.onboardingCompletedAt) {
        user.onboardingCompletedAt = new Date().toISOString();
        const welcomeConvId = randomUUID();
        if (user.letters.length === 0) {
          user.letters.push({
            id: randomUUID(),
            kind: 'personalized',
            title: 'نخستین پرتو آشنایی',
            salutation: `${user.displayName || 'دوست من'} عزیز،`,
            paragraphs: [
              'خوشحالم که این مسیر آرام را با هم آغاز کردیم. گاهی همین که به خودت اجازه بدهی مکث کنی و احساساتت را ببینی، آغاز روشنایی است.',
              'هر زمان که خواستی دربارهٔ چیزی که در ذهنت می‌گذرد حرف بزنی، من اینجا هستم تا بی‌عجله کنارت بمانم.',
            ],
            question: 'امروز چه چیزی بیش از همه در ذهنت حضور دارد؟',
            createdAt: new Date().toISOString(),
            sourceConversationIds: [welcomeConvId],
          });
        }
      }
      return sendJson(res, 200, {
        userId: user.userId,
        displayName: user.displayName,
        conversationStyle: user.conversationStyle,
        onboardingCompletedAt: user.onboardingCompletedAt,
      });
    }

    if (req.method === 'GET' && url.pathname === '/api/v1/me/onboarding-progress') {
      return sendJson(res, 200, { flowVersion: 'hamrah-v2', furthestStep: user.furthestStep });
    }

    if (req.method === 'PUT' && url.pathname === '/api/v1/me/onboarding-progress') {
      const body = await readJson(req);
      if (Number.isInteger(body.furthestStep) && body.furthestStep > user.furthestStep) {
        user.furthestStep = body.furthestStep;
      }
      return sendJson(res, 200, { flowVersion: 'hamrah-v2', furthestStep: user.furthestStep });
    }

    if (req.method === 'GET' && url.pathname === '/api/v1/me/privacy-preferences') {
      return sendJson(res, 200, { conversationSharingEnabled: user.privacySharing });
    }

    if (req.method === 'PUT' && url.pathname === '/api/v1/me/privacy-preferences') {
      const body = await readJson(req);
      if (typeof body.conversationSharingEnabled === 'boolean') {
        user.privacySharing = body.conversationSharingEnabled;
      }
      return sendJson(res, 200, { conversationSharingEnabled: user.privacySharing });
    }

    if (req.method === 'GET' && url.pathname === '/api/v1/me/letters') {
      return sendJson(res, 200, { items: user.letters });
    }

    if (req.method === 'GET' && url.pathname === '/api/v1/me/bookmarks') {
      return sendJson(res, 200, { items: Array.from(user.bookmarks.values()) });
    }

    const bookmarkMatch = url.pathname.match(/^\/api\/v1\/messages\/([0-9a-f-]+)\/bookmark$/i);
    if (bookmarkMatch) {
      const msgId = bookmarkMatch[1];
      if (req.method === 'DELETE') {
        user.bookmarks.delete(msgId);
        res.writeHead(204);
        res.end();
        return undefined;
      }
      if (req.method === 'PUT') {
        for (const conv of user.conversations) {
          const found = conv.messages.find((m) => m.id === msgId);
          if (found) {
            user.bookmarks.set(msgId, {
              ...found,
              conversationId: conv.id,
              savedAt: new Date().toISOString(),
            });
            return sendJson(res, 200, user.bookmarks.get(msgId));
          }
        }
        return sendJson(res, 404, { code: 'RESOURCE_NOT_FOUND', title: 'Message not found' });
      }
    }

    if (req.method === 'GET' && url.pathname === '/api/v1/conversations') {
      return sendJson(res, 200, {
        items: user.conversations.map(({ id, title, createdAt, updatedAt }) => ({ id, title, createdAt, updatedAt })),
      });
    }

    if (req.method === 'POST' && url.pathname === '/api/v1/conversations') {
      const now = new Date().toISOString();
      const created = { id: randomUUID(), title: 'گفت‌وگوی تازه', createdAt: now, updatedAt: now, messages: [] };
      user.conversations.unshift(created);
      return sendJson(res, 201, { id: created.id, title: created.title, createdAt: created.createdAt, updatedAt: created.updatedAt });
    }

    const convMatch = url.pathname.match(/^\/api\/v1\/conversations\/([0-9a-f-]+)(\/messages|\/turns)?$/i);
    if (convMatch) {
      const convId = convMatch[1];
      const sub = convMatch[2] || '';
      const conv = user.conversations.find((item) => item.id === convId);
      if (!conv) return sendJson(res, 404, { code: 'RESOURCE_NOT_FOUND', title: 'Not found' });

      if (!sub && req.method === 'PATCH') {
        const body = await readJson(req);
        conv.title = String(body.title || conv.title).trim().slice(0, 60);
        conv.updatedAt = new Date().toISOString();
        return sendJson(res, 200, { id: conv.id, title: conv.title, createdAt: conv.createdAt, updatedAt: conv.updatedAt });
      }
      if (!sub && req.method === 'DELETE') {
        user.conversations = user.conversations.filter((item) => item.id !== convId);
        for (const [k, v] of user.bookmarks.entries()) {
          if (v.conversationId === convId) user.bookmarks.delete(k);
        }
        res.writeHead(204);
        res.end();
        return undefined;
      }
      if (sub === '/messages' && req.method === 'GET') {
        return sendJson(res, 200, { items: conv.messages });
      }
      if (sub === '/turns' && req.method === 'POST') {
        const body = await readJson(req);
        const now = new Date().toISOString();
        const userMsg = { id: randomUUID(), role: 'user', content: String(body.message || ''), createdAt: now };
        const replyMsg = {
          id: randomUUID(),
          role: 'assistant',
          content: 'شنیدمت و ممنونم که این رو با من در میون گذاشتی.\n\nبیا یک لحظه با آرامش بهش نگاه کنیم: الان چه حسی دربارهٔ این موضوع در تو پررنگ‌تره؟',
          createdAt: new Date(Date.now() + 300).toISOString(),
        };
        conv.messages.push(userMsg, replyMsg);
        if (conv.title === 'گفت‌وگوی تازه') {
          conv.title = userMsg.content.split('\n')[0].slice(0, 28) || 'گفت‌وگو با تیا';
        }
        conv.updatedAt = replyMsg.createdAt;
        return sendJson(res, 200, { reply: replyMsg.content });
      }
    }

    return sendJson(res, 404, { code: 'NOT_FOUND', title: 'Endpoint not mocked in preview' });
  };
}

export async function startTheiaPreview({ host = '0.0.0.0', port = 5173 } = {}) {
  const apiMiddleware = createSyntheticPreviewApi();
  const server = await createServer({
    configFile: false,
    plugins: [
      react(),
      {
        name: 'theia-sandbox-preview-api',
        configureServer(vite) {
          vite.middlewares.use(apiMiddleware);
        },
      },
    ],
    server: {
      host,
      port,
      strictPort: true,
      allowedHosts: ['.e2b.app'],
    },
  });
  await server.listen();
  return server;
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const value = (flag, fallback) => {
    const index = process.argv.indexOf(flag);
    return index >= 0 ? process.argv[index + 1] : fallback;
  };
  const server = await startTheiaPreview({
    host: value('--host', '0.0.0.0'),
    port: Number(value('--port', '5173')),
  });
  server.printUrls();
  const close = async () => { await server.close(); process.exit(0); };
  process.on('SIGTERM', close);
  process.on('SIGINT', close);
}
