import React, { useCallback, useEffect, useState } from 'react';
import { OnboardingPath, Splash } from './screens/Onboarding.jsx';
import {
  NameScreen, LanguageScreen, RoadmapScreen, WelcomeScreen, RadioScreen,
  AgeScreen, SignupScreen, StatScreen, CountryScreen, FreeTextScreen,
  ReflectionScreen, SliderScreen, ScaleScreen, InterstitialScreen,
  CARRYING, EASED, LESS_HEAVY, MOVE_UP, YOURSELF, PERSON, FOUND, WHICH_AI, PHQ, GAD,
} from './screens/Intake.jsx';
import { GuideSelect, GuideDetail, Expressions, Duration } from './screens/Guides.jsx';
import { Session } from './screens/Session.jsx';
import { Home, Drawer, FocusDetail } from './screens/Home.jsx';
import { GUIDES } from './data.js';

export default function App() {
  const [stack, setStack] = useState([{ name: 'onboard' }]);
  const [drawer, setDrawer] = useState(false);
  const [toast, setToast] = useState(null);
  const [state, setState] = useState({
    name: 'مهدی',
    language: '',
    gender: '',
    age: '',
    email: '',
    country: '',
    countryQ: '',
    change: 5,
    about: '',
    brings: '',
    concerns: '',
    carrying: '',
    eased: '',
    lessHeavy: '',
    moveUp: '',
    yourself: '',
    person: '',
    found: '',
    whichAI: '',
    therapy: '',
    guide: GUIDES.find((g) => g.id === 'marcus'),
    hasFocus: false,
    duration: 20,
  });

  const set = useCallback((patch) => setState((s) => ({ ...s, ...patch })), []);
  const route = stack[stack.length - 1];

  const to = useCallback((name, params) => setStack((s) => [...s, { name, params }]), []);
  const back = useCallback(() => setStack((s) => (s.length > 1 ? s.slice(0, -1) : s)), []);
  const reset = useCallback((name) => setStack([{ name }]), []);

  const showToast = useCallback((msg) => {
    setToast({ msg, id: Date.now() });
    setTimeout(() => setToast((t) => (t && Date.now() - t.id >= 2600 ? null : t)), 2800);
  }, []);

  /* screenshot-detection toast (as in reference) */
  useEffect(() => {
    const h = (e) => { if (e.key === 'PrintScreen') showToast('رهراه این نماگرفت را شناسایی کرد'); };
    window.addEventListener('keyup', h);
    return () => window.removeEventListener('keyup', h);
  }, [showToast]);

  const g = state.guide;

  const screen = (() => {
    switch (route.name) {
      case 'onboard': return <OnboardingPath onNext={() => to('splash')} />;
      case 'splash': return <Splash onNext={() => to('name')} />;
      case 'name': return <NameScreen onBack={back} onNext={() => to('language')} state={state} set={set} />;
      case 'language': return <LanguageScreen onBack={back} onNext={() => to('roadmap')} state={state} set={set} />;
      case 'roadmap': return <RoadmapScreen onBack={back} onNext={() => to('welcome')} onSignin={() => to('signup')} />;
      case 'welcome': return <WelcomeScreen onNext={() => to('gender')} name={state.name} />;
      case 'gender': return <RadioScreen p={0.55} title="جنسیتت چیست؟" items={['مرد', 'زن', 'ترجیح می‌دهم توصیف کنم', 'ترجیح می‌دهم نگویم']} value={state.gender} onChange={(v) => set({ gender: v })} onBack={back} onNext={() => to('age')} />;
      case 'age': return <AgeScreen onBack={back} onNext={() => to('signup')} state={state} set={set} />;
      case 'signup': return <SignupScreen onBack={back} onNext={() => to('stat-orange')} state={state} set={set} />;
      case 'stat-orange': return <StatScreen k="orange" onBack={back} onNext={() => to('country')} />;
      case 'country': return <CountryScreen onBack={back} onNext={() => to('free-about')} state={state} set={set} />;
      case 'free-about': return <FreeTextScreen p={1.1} title="کمی از خودت بگو." sub="فعلاً مشکل را کنار بگذار — بیرون از آن، تو که هستی؟" placeholder="پاسخت را بنویس" hint="چه کسانی دورت هستند — و چه کسانی نیستند؟" skip value={state.about} onChange={(v) => set({ about: v })} onBack={back} onNext={() => to('refl-love')} />;
      case 'refl-love': return <ReflectionScreen k="love" onBack={back} onNext={() => to('therapy')} />;
      case 'therapy': return <RadioScreen p={1.15} title="تا به‌حال درمان یا کوچینگ تجربه کرده‌ای؟" items={['بله', 'نه', 'همین حالا هم در حال انجامم']} value={state.therapy} onChange={(v) => set({ therapy: v })} onBack={back} onNext={() => to('free-brings')} />;
      case 'free-brings': return <FreeTextScreen p={1.2} title="امروز چه چیزی آورده‌ت اینجا؟" sub="مشکلت، امیدت، آنچه در ذهنت است — هر چه هست." placeholder="چه چیزی آورده‌ت اینجا…" hint="چه چیزی امروز را روزش کرد؟" value={state.brings} onChange={(v) => set({ brings: v })} onBack={back} onNext={() => to('refl-hands')} />;
      case 'refl-hands': return <ReflectionScreen k="hands" onBack={back} onNext={() => to('free-concerns')} />;
      case 'free-concerns': return <FreeTextScreen p={1.25} title="نگرانی‌ای دربارهٔ اولین بار داری؟" sub="کاملاً طبیعی است که کمی داشته باشی. یا هیچ — آن هم خوب است." placeholder="اگر چیزی در ذهنت است، بنویس…" hint="چه چیزی این را امن می‌کند؟" skip value={state.concerns} onChange={(v) => set({ concerns: v })} onBack={back} onNext={() => to('carrying')} />;
      case 'carrying': return <RadioScreen p={1.35} title="کدام بهتر توصیف می‌کند آنچه بر دوش می‌کشی؟" items={CARRYING} value={state.carrying} onChange={(v) => set({ carrying: v })} onBack={back} onNext={() => to('slider')} />;
      case 'slider': return <SliderScreen onBack={back} onNext={() => to('refl-waiting')} state={state} set={set} />;
      case 'refl-waiting': return <ReflectionScreen k="waiting" onBack={back} onNext={() => to('stat-blue')} />;
      case 'stat-blue': return <StatScreen k="blue" onBack={back} onNext={() => to('eased')} />;
      case 'eased': return <RadioScreen p={1.45} title="اگر این کمی سبک‌تر شود، حتی خیلی کم، فکر می‌کنی چه چیزی جایش پیدا می‌شود؟" items={EASED} value={state.eased} onChange={(v) => set({ eased: v })} skip onBack={back} onNext={() => to('inter-sessions')} />;
      case 'inter-sessions': return <InterstitialScreen bg="bg-inter-orange" title="جلسه‌ها می‌توانند ۵ تا ۳۰ دقیقه باشند." sub="می‌توانی توقف کنی، کند بروی، یا مدتی هیچ نگویی. هیچ‌چیز اینجا آزمونی نیست که بشود در آن مردود شد." onBack={back} onNext={() => to('less-heavy')} />;
      case 'less-heavy': return <RadioScreen p={1.6} title="اخیراً روزی بود، حتی میان یک دورهٔ سخت، که کمی سبک‌تر احساس شود؟ چه چیزی فرق داشت؟" items={LESS_HEAVY} value={state.lessHeavy} onChange={(v) => set({ lessHeavy: v })} onBack={back} onNext={() => to('move-up')} />;
      case 'move-up': return <RadioScreen p={1.7} title="در جای امروزت، چه یک چیزی هست که حتی یک قدم هم بالاترت ببرد؟" items={MOVE_UP} value={state.moveUp} onChange={(v) => set({ moveUp: v })} onBack={back} onNext={() => to('stat-green62')} />;
      case 'stat-green62': return <StatScreen k="green62" onBack={back} onNext={() => to('yourself')} />;
      case 'yourself': return <RadioScreen p={1.75} title="به زمانی فکر کن، حتی کوتاه، که بیشترین شباهت را به خودت داشتی. چه کار می‌کردی، یا با که بودی؟" items={YOURSELF} value={state.yourself} onChange={(v) => set({ yourself: v })} onBack={back} onNext={() => to('person')} />;
      case 'person': return <RadioScreen p={1.8} title="حتی اگر چیزها مدتی سخت بمانند، می‌خواهی در میانهٔ آن چه جنس آدمی باشی؟" items={PERSON} value={state.person} onChange={(v) => set({ person: v })} onBack={back} onNext={() => to('inter-private')} />;
      case 'inter-private': return <InterstitialScreen bg="bg-inter-blue" title="اینجا جایی خصوصی و بی‌عجله است." sub="هر چه اینجا در میان می‌گذاری خصوصی و امن است و فقط برای چشمان تو." onBack={back} onNext={() => to('stat-green57')} />;
      case 'stat-green57': return <StatScreen k="green57" onBack={back} onNext={() => to('phq')} />;
      case 'phq': return <ScaleScreen p={2} statements={PHQ} onBack={back} onNext={() => to('found')} />;
      case 'found': return <RadioScreen p={3} title="رهراه را چطور پیدا کردی؟" items={FOUND} value={state.found} onChange={(v) => set({ found: v })} skip onBack={back} onNext={() => to('gad')} />;
      case 'gad': return <ScaleScreen p={2.5} statements={GAD} onBack={back} onNext={() => to('which-ai')} />;
      case 'which-ai': return <RadioScreen p={3} title="کدام یکی؟" items={WHICH_AI} value={state.whichAI} onChange={(v) => set({ whichAI: v })} skip onBack={back} onNext={() => { set({ hasFocus: false }); reset('home'); }} />;
      case 'home': return (
        <Home
          name={state.name}
          guide={g}
          hasFocus={state.hasFocus}
          onMenu={() => setDrawer(true)}
          onOpenFocus={() => to('focus')}
          onStartFocus={() => to('guides')}
        />
      );
      case 'guides': return (
        <GuideSelect
          onBack={back}
          onOpen={(gd) => to('guide-detail', { id: gd.id })}
          onSelect={(gd) => { set({ guide: gd }); to('expressions'); }}
          lockedToast={() => showToast('این راهنما فعلاً در دسترس نیست.')}
        />
      );
      case 'guide-detail': {
        const gd = GUIDES.find((x) => x.id === route.params.id);
        return <GuideDetail guide={gd} onClose={back} onSelect={(x) => { set({ guide: x }); to('expressions'); }} />;
      }
      case 'expressions': return <Expressions onClose={() => to('duration')} onAccept={() => to('duration')} onDecline={() => to('duration')} />;
      case 'duration': return <Duration onBack={back} onStart={(d) => { set({ duration: d }); to('session'); }} />;
      case 'session': return (
        <Session
          guide={g}
          name={state.name}
          duration={state.duration}
          onClose={() => { set({ hasFocus: true }); reset('home'); }}
          onPasteToast={() => showToast('رهراه از کلیپ‌بورد تو جای‌گذاری کرد')}
        />
      );
      case 'focus': return <FocusDetail guide={g} onBack={back} onStartSession={() => to('duration')} />;
      default: return null;
    }
  })();

  return (
    <div className="app">
      {screen}
      {drawer && route.name === 'home' && (
        <Drawer onClose={() => setDrawer(false)} onPast={() => setDrawer(false)} />
      )}
      {toast && <div className="toast" key={toast.id}>{toast.msg}</div>}
    </div>
  );
}
