import React, { useCallback, useEffect, useRef, useState } from 'react';
import { CheckCircle2 } from 'lucide-react';
import { FLOW, INITIAL_MESSAGES } from './data';
import OnboardingScreen from './screens/Onboarding';
import MainScreen from './screens/Main';

const EXTRA_ROUTES = ['home', 'survey', 'day', 'history', 'menu', 'chat', 'insight'];
const VALID_ROUTES = new Set([...FLOW, ...EXTRA_ROUTES]);
const DEFAULT_BACK = {
  home: 'welcome',
  survey: 'menu',
  day: 'home',
  history: 'home',
  menu: 'home',
  chat: 'home',
  insight: 'chat',
};

function routeFromHash() {
  const route = window.location.hash.replace(/^#\/?/, '').split('/')[0];
  return VALID_ROUTES.has(route) ? route : 'welcome';
}

export default function App() {
  const [route, setRoute] = useState(routeFromHash);
  const [name, setName] = useState('محمدمهدی');
  const [answers, setAnswers] = useState({});
  const [messages, setMessages] = useState(INITIAL_MESSAGES);
  const [toast, setToast] = useState('');
  const backOverrides = useRef({});

  useEffect(() => {
    if (!window.location.hash) window.history.replaceState(null, '', '#/welcome');
    const onHashChange = () => setRoute(routeFromHash());
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  }, []);

  useEffect(() => {
    window.scrollTo(0, 0);
    document.documentElement.style.colorScheme = ['home', 'survey', 'day', 'history', 'menu', 'chat', 'insight', 'research', 'experts', 'simple-intro', 'deeper-intro', 'know-intro', 'almost-intro', 'voice'].includes(route) ? 'dark' : 'light';
  }, [route]);

  useEffect(() => {
    if (!toast) return undefined;
    const timer = window.setTimeout(() => setToast(''), 2400);
    return () => window.clearTimeout(timer);
  }, [toast]);

  const navigate = useCallback((target) => {
    if (!VALID_ROUTES.has(target)) return;
    if (route === 'menu' && ['approach', 'privacy', 'research'].includes(target)) backOverrides.current[target] = 'menu';
    if (route === 'history' && target === 'day') backOverrides.current.day = 'history';
    if (route === 'day' && target === 'insight') backOverrides.current.insight = 'day';
    if (target === route) return;
    window.location.hash = `/${target}`;
  }, [route]);

  const back = useCallback(() => {
    const override = backOverrides.current[route];
    if (override) {
      delete backOverrides.current[route];
      navigate(override);
      return;
    }
    const flowIndex = FLOW.indexOf(route);
    if (flowIndex > 0) {
      navigate(FLOW[flowIndex - 1]);
      return;
    }
    navigate(DEFAULT_BACK[route] || 'home');
  }, [navigate, route]);

  const setAnswer = useCallback((key, value) => {
    setAnswers((current) => ({ ...current, [key]: value }));
  }, []);

  const notify = useCallback((message) => setToast(message), []);
  const reset = useCallback(() => {
    setAnswers({});
    setMessages(INITIAL_MESSAGES);
    setName('محمدمهدی');
    backOverrides.current = {};
  }, []);

  const shared = {
    route,
    navigate,
    back,
    name,
    setName,
    answers,
    setAnswer,
    messages,
    setMessages,
    notify,
    reset,
  };

  return (
    <div className="app-shell" dir="rtl">
      <main className="screen-transition" key={route}>
        {FLOW.includes(route) && route !== 'home' ? <OnboardingScreen {...shared} /> : <MainScreen {...shared} />}
      </main>
      {toast && <div className="toast" role="status"><CheckCircle2 size={19} /><span>{toast}</span></div>}
    </div>
  );
}
