import React, { useEffect, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import '@fontsource-variable/estedad';
import '@fontsource-variable/vazirmatn';
import '@fontsource-variable/noto-naskh-arabic';
import '../../src/styles.css';
import OnboardingScreen from '../../src/screens/Onboarding';
import MainScreen from '../../src/screens/Main';
import DevLogin from '../../src/features/devsession/DevLogin';
import { screenSurfaceForRoute } from '../../src/features/appearance/screenSurface';
import { KenarCompanion, KenarGlyph, KenarMark, KenarWordmark } from '../../src/components/Kenar';

/**
 * Isolated identity preview. It mounts the real screens with synthetic props only:
 * no session, no API call, no persisted data. Everything here is illustrative.
 */

const noop = () => {};
const ANSWERS = {
  approach: 'همدلی و شنیدن',
  voice: 'آرام',
  mood: 3,
  concerns: ['کار یا درس'],
  focus: ['خودم رو بهتر بشناسم'],
  recharge: ['رفتن به دل طبیعت'],
};

const LETTERS = [
  {
    id: '00000000-0000-4000-8000-000000000001',
    title: 'دربارهٔ روزهایی که سنگین‌اند',
    createdAt: new Date().toISOString(),
    salutation: 'دوست من،',
    paragraphs: [
      'این روزها شاید حس کنی همه‌چیز زیادی نزدیک است؛ کار، فکرها، حتی سکوت. لازم نیست همین حالا چیزی را مرتب کنی.',
      'یک کار کوچک کافی است: یک نفس، یک لیوان آب، یا فقط نشستن. من همین‌جا می‌مانم تا وقتی که بخواهی حرف بزنی.',
    ],
    question: 'اگر امروز فقط یک چیز را سبک‌تر کنی، کدام چیز است؟',
  },
  {
    id: '00000000-0000-4000-8000-000000000002',
    title: 'وقتی کسی را کنارِ خودت می‌خواهی',
    createdAt: new Date(Date.now() - 86400000).toISOString(),
    salutation: 'دوست من،',
    paragraphs: ['بعضی روزها فقط باید گفته شود؛ نه راه‌حل، نه جواب.'],
    question: null,
  },
];

const MESSAGES = [
  {
    id: 'm-1', role: 'user', createdAt: new Date().toISOString(),
    content: 'این روزها همه‌چیز خیلی سنگین شده. نمی‌دونم از کجا شروع کنم.',
  },
  {
    id: 'm-2', role: 'assistant', createdAt: new Date().toISOString(),
    content: 'لازم نیست از جایی شروع کنی.\n\nهر چیزی که همین حالا توی ذهنت هست را بگو؛ همان کافی است. من همین‌جا می‌مانم.',
  },
  {
    id: 'm-3', role: 'user', createdAt: new Date().toISOString(),
    content: 'فکر می‌کنم فقط خسته‌ام.',
  },
  {
    id: 'm-4', role: 'assistant', createdAt: new Date().toISOString(),
    content: 'خستگی هم چیزی است که می‌شود درباره‌اش حرف زد.\n\nچه چیزی بیشتر از همه انرژی می‌گیرد؟',
  },
];

function Frame({ label, route, children, frameRef }) {
  return (
    <div className="bp-item" ref={frameRef}>
      <div className="bp-frame" data-screen-surface={screenSurfaceForRoute(route)}>
        <div className="app-shell" dir="rtl" data-screen-surface={screenSurfaceForRoute(route)}>
          <main className="screen-transition">{children}</main>
        </div>
      </div>
      <div className="bp-label">
        <strong>{label}</strong>
        <span>#{route}</span>
      </div>
    </div>
  );
}

function OnboardingFrame({ label, route }) {
  const [name, setName] = useState('آوا');
  const [answers, setAnswers] = useState(ANSWERS);
  const setAnswer = (key, value) => setAnswers((current) => ({ ...current, [key]: value }));
  return (
    <Frame label={label} route={route}>
      <OnboardingScreen
        route={route}
        navigate={noop}
        back={noop}
        name={name}
        setName={setName}
        answers={answers}
        setAnswer={setAnswer}
        notify={noop}
        privacySharing
        privacyLoaded
        privacyLoading={false}
        privacySaving={false}
        onTogglePrivacy={noop}
        onRetryPrivacy={noop}
        savingProfile={false}
        profileEdit={false}
        onSaveName={noop}
        onSaveOnboardingName={noop}
        onSaveApproach={noop}
      />
    </Frame>
  );
}

function HomeFrame() {
  return (
    <Frame label="خانه" route="home">
      <MainScreen
        route="home"
        navigate={noop}
        name="آوا"
        letterPhase="ready"
        letters={LETTERS}
        readLetterIds={[LETTERS[1].id]}
        onNewConversation={() => true}
      />
    </Frame>
  );
}

function ChatFrame() {
  const [draft, setDraft] = useState('');
  return (
    <Frame label="گفت‌وگو" route="chat">
      <MainScreen
        route="chat"
        navigate={noop}
        back={noop}
        messages={MESSAGES}
        conversations={[{ id: 'c-1', title: 'دربارهٔ خستگی', createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() }]}
        conversationId="c-1"
        activeConversation={{ id: 'c-1', title: 'دربارهٔ خستگی' }}
        pendingTurn={null}
        sending={false}
        chatLoading={false}
        chatError={null}
        historyPhase="ready"
        historyError={null}
        chatDraft={draft}
        onDraftChange={setDraft}
        onSendMessage={() => false}
        onRetryMessage={noop}
        onRetryChatLoad={noop}
        onNewConversation={() => true}
        onOpenConversation={() => true}
        onDeleteConversation={noop}
        onRenameConversation={() => true}
        onReloadHistory={noop}
        bookmarks={[]}
        bookmarkPhase="ready"
        bookmarkError={null}
        bookmarkBusyId={null}
        onToggleBookmark={() => false}
        onReloadBookmarks={noop}
        savedMessageTarget={null}
        onSavedMessageTargetDone={noop}
        notify={noop}
        onboardingGreeting=""
        letterReply={null}
        onDismissLetterReply={noop}
      />
    </Frame>
  );
}

function LettersFrame({ label, route }) {
  return (
    <Frame label={label} route={route}>
      <MainScreen
        route={route}
        back={noop}
        navigate={noop}
        letterPhase="ready"
        letters={LETTERS}
        readLetterIds={[LETTERS[1].id]}
        onReadLetter={noop}
        onReloadLetters={noop}
        onDiscussLetter={noop}
      />
    </Frame>
  );
}

function LetterReadingFrame() {
  const frameRef = useRef(null);
  useEffect(() => {
    // Preview only: open the first letter so the reader (and its signature) is visible.
    const timer = setTimeout(() => frameRef.current?.querySelector('.letter-row')?.click(), 400);
    return () => clearTimeout(timer);
  }, []);
  return (
    <div ref={frameRef}>
      <LettersFrame label="نامه · خواندن" route="letter" />
    </div>
  );
}

function MenuFrame() {
  return (
    <Frame label="منو" route="menu">
      <MainScreen
        route="menu"
        navigate={noop}
        name="آوا"
        letterPhase="ready"
        letters={LETTERS}
        readLetterIds={[LETTERS[1].id]}
        onNewConversation={() => true}
        onLogout={noop}
        onSwitchTester={noop}
      />
    </Frame>
  );
}

function LoginFrame() {
  return (
    <Frame label="ورود (محیط آزمایشی)" route="welcome">
      <DevLogin session={{ phase: 'unauthenticated', error: null }} login={noop} retry={noop} />
    </Frame>
  );
}

function BrandSheet() {
  return (
    <div className="bp-wrap">
      <p className="bp-eyebrow">هویت بصری · ۲۰۲۶</p>
      <h1 className="bp-title">کنار</h1>
      <p className="bp-lead">
        کنار یعنی همین‌جا بودن، کنارِ تو. هویت تازه از یک خط ساخته می‌شود — خطِ گفت‌وگو — و یک حضور
        کوچک که روی همان خط، کنارِ واژه‌ها می‌نشیند.
      </p>
      <p className="bp-note">
        <b>پیش‌نمایش طراحی</b>
        داده‌ها ساختگی‌اند؛ هیچ اتصالی به سرور یا حساب واقعی وجود ندارد.
      </p>

      <section className="bp-section">
        <h2>نشان و لوگوتایپ</h2>
        <p>همان حضور، در قالب یک نشان: گنبدِ شنی، دو نقطه و یک لبخند، نشسته روی افق.</p>
        <div className="bp-lockup">
          <div className="bp-card bp-card--paper">
            <img src="/brand/kenar-lockup-light.png" alt="لوگوتایپ کنار در زمینهٔ روشن" />
          </div>
          <div className="bp-card">
            <img src="/brand/kenar-lockup-dark.png" alt="لوگوتایپ کنار در زمینهٔ تیره" />
          </div>
        </div>
        <div className="bp-lockup" style={{ marginTop: 20 }}>
          <div className="bp-card bp-card--paper bp-card-pad">
            <h3>نشان در اندازه‌های واقعی</h3>
            <small>۱۶ تا ۴۸ پیکسل — برای آواتار گفت‌وگو، مهر نامه و نشان‌های ریز</small>
            <div style={{ display: 'flex', alignItems: 'flex-end', gap: 16, marginTop: 18, color: '#23201b' }}>
              {[16, 20, 24, 32, 40, 48].map((size) => <KenarMark key={size} size={size} />)}
            </div>
          </div>
          <div className="bp-card bp-card-pad">
            <h3>آیکون برنامه</h3>
            <small>روی پس‌زمینهٔ شب، با هالهٔ گرم و خط دوم افق</small>
            <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginTop: 18 }}>
              <img src="/brand/kenar-icon-120.png" alt="آیکون کنار" width="88" height="88" style={{ borderRadius: 20 }} />
              <img src="/brand/kenar-icon-64.png" alt="" width="44" height="44" style={{ borderRadius: 11 }} onError={(event) => { event.currentTarget.style.display = 'none'; }} />
              <img src="/brand/favicon-32.png" alt="" width="26" height="26" style={{ borderRadius: 7 }} />
            </div>
          </div>
        </div>
      </section>

      <section className="bp-section">
        <h2>شخصیت</h2>
        <p>هم‌نشین: گنبدی نرم با یک توئکِ کوچک روی سر، دو چشم و یک لبخند آرام. دست و پا ندارد؛ کاری با تو نمی‌کند، فقط می‌ماند.</p>
        <div className="bp-posese">
          {[['sit', 'نشسته', 'اولین آشنایی'], ['lean', 'تکیه‌داده', 'گوش می‌دهم'], ['rest', 'چشم‌بسته', 'بدون فشار'], ['voice', 'صدا', 'حالا بشنو'], ['peek', 'نیمه‌پیدا', 'کنارِ صفحه']].map(([pose, title, note]) => (
            <div className="bp-pose" key={pose} style={{ color: '#23201b' }}>
              <KenarCompanion pose={pose} animated={false} />
              <strong>{title}</strong>
              <span>{note}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="bp-section">
        <h2>رنگ</h2>
        <p>پالت فعلی محصول دست‌نخورده است؛ کنار فقط سه رنگِ مربوط به خودش را اضافه می‌کند.</p>
        <div className="bp-swatches">
          {[
            ['#e9d0ae', '—kenar-sand', 'بدنِ هم‌نشین'],
            ['#b9744f', '—kenar-clay', 'سرخیِ گونه، مهر'],
            ['#a89e90', '—kenar-shore', 'خطِ دومِ کم‌رنگ'],
            ['#12110d', '—dark', 'شبِ محصول'],
            ['#f3f2ee', '—light', 'کاغذِ روشن'],
            ['#cec4b8', '—beige', 'ساحلِ میانی'],
          ].map(([hex, name, note]) => (
            <div className="bp-swatch" key={name}>
              <i style={{ background: hex, border: hex === '#f3f2ee' ? '1px solid #2d2a26' : 'none' }} />
              <b>{name}</b>
              <span>{note} · {hex}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="bp-section">
        <h2>تایپوگرافی و امضای کلمه</h2>
        <p>فونت‌های فعلی می‌مانند: Estedad برای رابط، Noto Naskh برای نامه‌ها.</p>
        <div className="bp-grid-2">
          <div className="bp-card bp-card--paper bp-card-pad" style={{ color: '#23201b' }}>
            <KenarWordmark />
            <div className="bp-typography" style={{ marginTop: 26 }}>
              <div className="bp-type-row"><strong style={{ fontSize: 30, fontWeight: 730, letterSpacing: '-.9px' }}>با کنار آشنا شو</strong><span>Page header · 30/730</span></div>
              <div className="bp-type-row"><strong style={{ fontSize: 17, fontWeight: 400 }}>اینجا می‌تونی راحت حرف بزنی.</strong><span>Body · 17/400</span></div>
              <div className="bp-type-row"><strong style={{ fontSize: 21, fontFamily: "'Noto Naskh Arabic Variable', serif", fontWeight: 500 }}>نامه‌های کنار</strong><span>Naskh · letters</span></div>
            </div>
          </div>
          <div className="bp-card bp-card-pad">
            <h3>جمله‌های آغاز رابطه</h3>
            <ul className="bp-story" style={{ margin: '16px 0 0', padding: 0 }}>
              {[
                ['۱', 'سلام. من کنارم.', 'اولین جملهٔ هم‌نشین، زیرِ تصویر خوش‌آمد.'],
                ['۲', 'همین‌جا می‌مونم؛ هر وقت آماده بودی.', 'در میانهٔ پرسش‌های ساده.'],
                ['۳', 'لازم نیست همه‌چیز رو یه‌بار بگی.', 'وقتی پرسش‌ها عمیق‌تر می‌شوند.'],
                ['۴', 'داری برام آشنا می‌شی.', 'چشم‌ها بسته، بی‌عجله.'],
                ['۵', 'از این به بعد، کنارتم.', 'صفحهٔ آماده‌بودن؛ شروع گفت‌وگو.'],
              ].map(([n, line, note]) => (
                <li key={n}>
                  <em>{n}</em>
                  <div><b>{line}</b><p>{note}</p></div>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </section>

      <section className="bp-section">
        <h2>در متنِ محصول</h2>
        <p>همان چیدمان‌ها، همان ناوبری و همان تعامل‌ها؛ فقط نشانه‌ها و تصویرها عوض شده‌اند. تصویرها را افقی بکشید.</p>
        <div className="bp-gallery">
          <OnboardingFrame label="خوش‌آمد" route="welcome" />
          <OnboardingFrame label="آشنایی · قدم اول" route="simple-intro" />
          <OnboardingFrame label="آشنایی · عمیق‌تر" route="deeper-intro" />
          <OnboardingFrame label="آشنایی · چشم‌بسته" route="know-intro" />
          <OnboardingFrame label="نزدیکِ صدا" route="almost-intro" />
          <OnboardingFrame label="حریم خصوصی" route="privacy" />
          <OnboardingFrame label="یادآوری" route="checkins" />
          <OnboardingFrame label="آماده" route="ready" />
          <OnboardingFrame label="سبک گفت‌وگو" route="approach" />
          <HomeFrame />
          <ChatFrame />
          <LettersFrame label="نامه‌ها" route="letter" />
          <LetterReadingFrame />
          <MenuFrame />
          <LoginFrame />
        </div>
      </section>

      <section className="bp-section">
        <h2>حفظ‌شده</h2>
        <div className="bp-grid-2">
          <div className="bp-card bp-card-pad">
            <h3>چیزی که تغییر نکرد</h3>
            <ul className="bp-story" style={{ margin: '16px 0 0', padding: 0, color: '#9a948a' }}>
              <li><em>✓</em><div><b>ساختار و چیدمان</b><p>همان صفحات، همان فاصله‌ها، همان دکمه‌ها و مسیر ۲۶ مرحله‌ای.</p></div></li>
              <li><em>✓</em><div><b>ناوبری و تعامل</b><p>سوایپ، چرخِ سن، اسلایدر حال، شیت‌ها و حالت‌های خطا دست‌نخورده‌اند.</p></div></li>
              <li><em>✓</em><div><b>دسترس‌پذیری</b><p>تصویرها تزئینی‌اند؛ متن‌ها واقعی؛ حرکت با کاهش حرکت خاموش می‌شود.</p></div></li>
            </ul>
          </div>
          <div className="bp-card bp-card-pad">
            <h3>چیزی که عوض شد</h3>
            <ul className="bp-story" style={{ margin: '16px 0 0', padding: 0, color: '#9a948a' }}>
              <li><em>←</em><div><b>گل‌ها رفتند</b><p>هیچ گل، گلبرگ، دیزی یا نماد گیاهی باقی نمانده است.</p></div></li>
              <li><em>←</em><div><b>حضور آمد</b><p>هم‌نشین، افق، خط دوم و مهرِ موم جای آن‌ها را گرفت.</p></div></li>
              <li><em>←</em><div><b>اسم، همه‌جا</b><p>عنوان مرورگر، آیکون، منو، نامه‌ها و اولین پیام گفت‌وگو.</p></div></li>
            </ul>
          </div>
        </div>
      </section>
    </div>
  );
}

const container = document.getElementById('brand-preview');
if (container && container.nodeType === 1) createRoot(container).render(<BrandSheet />);

export { BrandSheet };
