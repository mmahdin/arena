import React, { useEffect, useState } from 'react';
import { createRoot } from 'react-dom/client';
import '@fontsource-variable/estedad';
import '@fontsource-variable/vazirmatn';
import '@fontsource-variable/noto-naskh-arabic';
import OnboardingScreen from '../../src/screens/Onboarding';
import '../../src/styles.css';

/**
 * Isolated, synthetic design preview. It renders the real screen component but
 * never mounts App, authenticates a tester, or calls the conversation/profile API.
 * Use ?edit=1 to exercise the completed-profile editor instead of onboarding.
 */
function ApproachPreview() {
  const params = new URLSearchParams(window.location.search);
  const profileEdit = params.get('edit') === '1';
  const [answers, setAnswers] = useState({});
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState('');

  useEffect(() => {
    if (!notice) return undefined;
    const timer = window.setTimeout(() => setNotice(''), 2600);
    return () => window.clearTimeout(timer);
  }, [notice]);

  const save = (approach) => {
    setSaving(true);
    window.setTimeout(() => {
      setSaving(false);
      setNotice(`پیش‌نمایش: «${approach}» انتخاب شد؛ ذخیرهٔ سروری انجام نشد.`);
    }, 450);
  };

  return (
    <div className="app-shell" data-screen-surface="beige" dir="rtl">
      <p className="sr-only">پیش‌نمایش طراحی با دادهٔ ساختگی؛ این صفحه به سرور متصل نیست.</p>
      <main className="screen-transition">
        <OnboardingScreen
          route="approach"
          answers={answers}
          setAnswer={(key, value) => setAnswers((current) => ({ ...current, [key]: value }))}
          navigate={(route) => setNotice(`پیش‌نمایش: مسیر بعدی ${route}`)}
          back={() => setNotice('پیش‌نمایش: بازگشت')}
          profileEdit={profileEdit}
          savingProfile={saving}
          onSaveApproach={save}
        />
      </main>
      {notice && <div className="toast" role="status">{notice}</div>}
    </div>
  );
}

createRoot(document.getElementById('root')).render(
  <React.StrictMode><ApproachPreview /></React.StrictMode>,
);
