import React, { useState } from 'react';
import { RefreshCw } from 'lucide-react';
import { PrimaryButton, StatusBar } from '../../components/UI';
import { LightStudy, TheiaMark, TheiaWordmark } from '../../components/Brand';
import { normalizeTestPhone } from './phone';

/** Phone-shaped entry for private test environments, dressed as Theia's front door. */
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
      <div className="dev-login-brand">
        <span className="welcome-brand__mark">
          <TheiaMark size={30} color="var(--text-ink)" />
          <TheiaWordmark size={27} tone="ink" />
        </span>
        <span className="welcome-brand__tag">همراهی برای دیدنِ روشن‌تر</span>
      </div>
      <div className="dev-login-art" aria-hidden="true">
        <LightStudy scene="horizon" />
      </div>
      <div className="dev-login-panel">
        <div className="dev-login-heading">
          <span className="eyebrow">ورود</span>
        </div>
        <p>برای ادامه، شمارهٔ موبایلت رو وارد کن.</p>
        {loading && <p role="status">در حال ورود…</p>}
        {offline && <div className="dev-login-error" role="alert">
          <p>{session.error?.action === 'logout' ? 'خروج انجام نشد. دوباره تلاش کن.' : 'ارتباط برقرار نشد. دوباره تلاش کن.'}</p>
          <button className="dev-login-retry" type="button" onClick={retry}><RefreshCw size={15} aria-hidden="true" />تلاش دوباره</button>
        </div>}
        <form className="dev-phone-form" onSubmit={submit}>
          <label htmlFor="dev-phone">شماره موبایل</label>
          <input id="dev-phone" dir="ltr" type="text" inputMode="numeric" autoComplete="off" spellCheck="false" maxLength={11} placeholder="09xxxxxxxxx" value={phone}
            onChange={(event) => { setPhone(event.target.value); setAttempted(false); }} disabled={loading || offline} aria-invalid={invalid} />
          {(attempted || invalid) && !normalized && <div className="dev-login-error" role="alert">شماره باید ۱۱ رقم باشه و با ۰۹ شروع بشه.</div>}
          {session.error?.action === 'login' && !offline && <div className="dev-login-error" role="alert">ورود انجام نشد. دوباره تلاش کن.</div>}
          {!offline && <PrimaryButton type="submit" disabled={loading || !normalized}>{loading ? 'کمی صبر کن…' : 'ادامه'}</PrimaryButton>}
        </form>
      </div>
    </section>
  );
}
