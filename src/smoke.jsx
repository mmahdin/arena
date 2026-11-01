import React from 'react';
import { renderToString } from 'react-dom/server';
import { OnboardingPath, Splash } from './screens/Onboarding.jsx';
import {
  NameScreen, LanguageScreen, RoadmapScreen, WelcomeScreen, RadioScreen,
  AgeScreen, SignupScreen, StatScreen, CountryScreen, FreeTextScreen,
  ReflectionScreen, SliderScreen, ScaleScreen, InterstitialScreen,
} from './screens/Intake.jsx';
import { GuideSelect, GuideDetail, Expressions, Duration } from './screens/Guides.jsx';
import { Session } from './screens/Session.jsx';
import { Home, Drawer, FocusDetail } from './screens/Home.jsx';
import { GUIDES, CARRYING } from './data.js';

const noop = () => {};
const state = { name: 'مهدی', language: '', gender: '', age: '', email: '', country: '', countryQ: '', change: 5, about: '', brings: '', concerns: '', carrying: '', eased: '', therapy: '' };
const set = noop;
const g = GUIDES.find((x) => x.id === 'marcus');

const cases = [
  ['OnboardingPath', <OnboardingPath onNext={noop} />],
  ['Splash', <Splash onNext={noop} />],
  ['Name', <NameScreen onBack={noop} onNext={noop} state={state} set={set} />],
  ['Language', <LanguageScreen onBack={noop} onNext={noop} state={state} set={set} />],
  ['Roadmap', <RoadmapScreen onBack={noop} onNext={noop} onSignin={noop} />],
  ['Welcome', <WelcomeScreen onNext={noop} name="مهدی" />],
  ['Radio', <RadioScreen p={1} title="t" items={CARRYING} value="" onChange={noop} onBack={noop} onNext={noop} />],
  ['Age', <AgeScreen onBack={noop} onNext={noop} state={state} set={set} />],
  ['Signup', <SignupScreen onBack={noop} onNext={noop} state={state} set={set} />],
  ['StatOrange', <StatScreen k="orange" onBack={noop} onNext={noop} />],
  ['StatBlue', <StatScreen k="blue" onBack={noop} onNext={noop} />],
  ['Country', <CountryScreen onBack={noop} onNext={noop} state={state} set={set} />],
  ['FreeText', <FreeTextScreen p={1} title="t" placeholder="p" hint="h" skip value="" onChange={noop} onBack={noop} onNext={noop} />],
  ['Reflection', <ReflectionScreen k="love" onBack={noop} onNext={noop} />],
  ['Slider', <SliderScreen onBack={noop} onNext={noop} state={state} set={set} />],
  ['Scale', <ScaleScreen p={2} statements={['a', 'b']} onBack={noop} onNext={noop} />],
  ['Interstitial', <InterstitialScreen bg="bg-inter-blue" title="t" sub="s" onBack={noop} onNext={noop} />],
  ['GuideSelect', <GuideSelect onBack={noop} onOpen={noop} onSelect={noop} lockedToast={noop} />],
  ['GuideDetail', <GuideDetail guide={g} onClose={noop} onSelect={noop} />],
  ['Expressions', <Expressions onClose={noop} onAccept={noop} onDecline={noop} />],
  ['Duration', <Duration onBack={noop} onStart={noop} />],
  ['Session', <Session guide={g} name="مهدی" duration={20} onClose={noop} onPasteToast={noop} />],
  ['HomeEmpty', <Home name="مهدی" guide={g} hasFocus={false} onMenu={noop} onOpenFocus={noop} onStartFocus={noop} />],
  ['HomeCard', <Home name="مهدی" guide={g} hasFocus onMenu={noop} onOpenFocus={noop} onStartFocus={noop} />],
  ['Drawer', <Drawer onClose={noop} onPast={noop} />],
  ['Focus', <FocusDetail guide={g} onBack={noop} onStartSession={noop} />],
];

let fail = 0;
for (const [name, el] of cases) {
  try {
    const html = renderToString(el);
    console.log('OK  ', name, html.length);
  } catch (e) {
    fail++;
    console.log('FAIL', name, e.message);
  }
}
process.exit(fail ? 1 : 0);
