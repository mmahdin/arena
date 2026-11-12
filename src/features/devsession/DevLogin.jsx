import React, { useState } from 'react';
import { RefreshCw } from 'lucide-react';
import { TheiaLoginArt, TheiaMark } from '../../components/TheiaBrand';
import { PageHeader, PrimaryButton, StatusBar } from '../../components/UI';
import { normalizeTestPhone } from './phone';

/** Simple phone-shaped entry; backend runs only in private dev environments. */
export default function DevLogin({ session, login, retry }) {
  const [phone, setPhone] = useState('');
  const [attempted, setAttempted] = useState(false);
  const normalized = normalizeTestPhone(phone);
  const invalid = phone.length >= 11 && !normalized;
  const loading = session.phase === 'loading' || session.phase === 'working';
  const offline = session.phase === 'error';
  const submit = (event) => {
    event.preventDefault();
    setAttempted(true);
    if (normalized && !loading && !offline) login(normalized);
  };
  return (
    <section className="app-screen theme-light dev-login-screen" aria-busy={loading}>
      <StatusBar />
      <PageHeader title="به تیا خوش اومدی!" />
      <div className="dev-login-art" aria-hidden="true"><TheiaLoginArt /></div>
      <div className="dev-login-panel">
        <div className="dev-login-heading"><TheiaMark size={21} /><strong>ورود به حساب</strong></div>
        {loading && <p role="status">در حال ورود…</p>}
        {offline && <div className="dev-login-error" role="alert">
          <p>{session.error?.action === 'logout' ? 'خروج انجام نشد. دوباره تلاش کن.' : 'ارتباط برقرار نشد. دوباره تلاش کن.'}</p>
          <button className="dev-login-retry" type="button" onClick={retry}><RefreshCw size={18} aria-hidden="true" />تلاش دوباره</button>
        </div>}
        <form className="dev-phone-form" onSubmit={submit}>
          <label htmlFor="dev-phone">شماره موبایل</label>
          <input id="dev-phone" dir="ltr" type="text" inputMode="numeric" autoComplete="off" spellCheck="false" maxLength={11} placeholder="09xxxxxxxxx" value={phone}
            onChange={(event) => { setPhone(event.target.value); setAttempted(false); }} disabled={loading || offline} aria-invalid={invalid} />
          {(attempted || invalid) && !normalized && <div className="dev-login-error" role="alert">شماره باید ۱۱ رقم باشه و با ۰۹ شروع بشه.</div>}
          {session.error?.action === 'login' && !offline && <div className="dev-login-error" role="alert">ورود انجام نشد. دوباره تلاش کن.</div>}
          {!offline && <PrimaryButton type="submit" disabled={loading || !normalized}>ادامه</PrimaryButton>}
        </form>
      </div>
    </section>
  );
}
