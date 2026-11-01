import React from 'react';
import { FOCUS } from '../data.js';
import { Pill, faNum, BRAND } from '../components/UI.jsx';
import { Menu, Sprout, History, Stack, Person, Back, Calendar, Clock, Mark } from '../components/Icons.jsx';

export function Home({ name, guide, hasFocus, onMenu, onOpenFocus, onStartFocus }) {
  return (
    <div className="screen home no-anim" style={{ background: 'var(--light)' }}>
      <div className="hero bg-home-warm">
        <button className="menu" type="button" onClick={onMenu} aria-label="منو"><Menu /></button>
        <div className="greet">عصر بخیر، {name || 'دوست عزیز'}</div>
      </div>
      <div className="body">
        {hasFocus ? (
          <div className="fcard" onClick={onOpenFocus}>
            <div className="row1">
              <h3>{FOCUS.title}</h3>
              <img className="av" src={guide.img} alt={guide.name} />
            </div>
            <p>{FOCUS.excerpt}</p>
            <div className="row2">
              <span className="chip-tag">{FOCUS.tag}</span>
              <span className="ago"><History size={19} /> {FOCUS.ago}</span>
            </div>
          </div>
        ) : (
          <div className="fcard emptycard">
            <span className="ic"><Sprout /></span>
            <h3>اولین حوزهٔ تمرکزت را شروع کن</h3>
            <p>حوزه‌های تمرکز به تو و راهنمایت جا می‌دهند تا با هم از میان چالش‌ها بگذرید.</p>
          </div>
        )}
      </div>
      <div className="btn-row">
        <Pill variant="dark" onClick={onStartFocus}>شروع یک حوزهٔ تمرکز تازه</Pill>
      </div>
    </div>
  );
}

export function Drawer({ onClose, onPast }) {
  return (
    <>
      <div className="drawer-scrim" onClick={onClose} />
      <div className="drawer">
        <span className="brand"><Mark size={30} sw={6} /> {BRAND}</span>
        <nav>
          <button className="item active" type="button" onClick={onClose}><Stack /> حوزه‌های تمرکز</button>
          <button className="item" type="button" onClick={onPast}><History /> جلسه‌های گذشته</button>
        </nav>
        <div className="spacer" />
        <button className="item" type="button"><Person /> تنظیمات</button>
      </div>
    </>
  );
}

export function FocusDetail({ guide, onBack, onStartSession }) {
  return (
    <div className="screen focus" style={{ background: 'var(--light)' }}>
      <div className="hero bg-focus-pink">
        <button className="back" type="button" onClick={onBack} aria-label="بازگشت"><Back size={22} /></button>
        <h2>{FOCUS.title}</h2>
        <p>{FOCUS.excerpt}</p>
      </div>
      <div className="body">
        <div className="review">
          <h3>در حال بررسی جلسه‌ات</h3>
          <div className="dots3"><i /><i /><i /></div>
          <p>کمی بعد سر بزن</p>
        </div>
        <p className="eyebrow" style={{ color: '#55585b' }}>گزارش‌های جلسه</p>
        <div className="report">
          <div className="r1">
            <span><Calendar /> {FOCUS.report.date}</span>
            <span><Clock /> {FOCUS.report.dur}</span>
          </div>
          <h4>{FOCUS.report.title}</h4>
          <p>{FOCUS.report.excerpt}</p>
        </div>
        <div className="archive">
          <Pill variant="outline">بایگانی کردن حوزهٔ تمرکز</Pill>
        </div>
        <div className="mind">
          <div className="row">
            <img src={guide.img} alt={guide.name} />
            <span className="q">چه چیزی در ذهنت است؟</span>
          </div>
          <Pill variant="light" onClick={onStartSession}>شروع جلسه</Pill>
        </div>
      </div>
    </div>
  );
}
