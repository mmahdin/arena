import React, { useEffect, useRef, useState } from 'react';
import { GUIDES } from '../data.js';
import { Pill, TopBar, faNum } from '../components/UI.jsx';
import { Back, Sliders, Lock, Waveform, X, Eye, Camera, Video, CheckCircle } from '../components/Icons.jsx';

/* ---------- choose guide ---------- */
export function GuideSelect({ onBack, onOpen, onSelect, lockedToast }) {
  const [active, setActive] = useState(GUIDES[0]);
  const refs = useRef({});

  useEffect(() => {
    const obs = new IntersectionObserver(
      (ents) => {
        ents.forEach((en) => {
          if (en.isIntersecting) setActive(GUIDES.find((g) => g.id === en.target.dataset.g));
        });
      },
      { threshold: 0.6 },
    );
    Object.values(refs.current).forEach((el) => el && obs.observe(el));
    return () => obs.disconnect();
  }, []);

  return (
    <div className="screen guides">
      <div className="topbar on-dark" style={{ flex: 'none' }}>
        <button className="iconbtn" onClick={onBack} aria-label="بازگشت" type="button"><Back /></button>
        <span className="brand" style={{ fontSize: 27 }}>راهنمای هوش مصنوعی‌ات را انتخاب کن</span>
        <span className="iconbtn end"><Sliders /></span>
      </div>
      <div className="list">
        {GUIDES.map((g) => (
          <div
            key={g.id}
            className="gcard"
            data-g={g.id}
            ref={(el) => (refs.current[g.id] = el)}
            onClick={() => onOpen(g)}
          >
            <img src={g.img} alt={g.name} />
            <div className="shade" />
            <div className="info">
              <h3 className="name">
                {g.locked && <Lock size={26} />}
                {g.name}
              </h3>
              <p className="desc">{g.desc}</p>
              <div className="tags">
                {g.tags.map((t) => <span className="gtag" key={t}>{t}</span>)}
              </div>
            </div>
            <button className="voice" type="button" aria-label="شنیدن صدا" onClick={(e) => e.stopPropagation()}>
              <Waveform />
            </button>
          </div>
        ))}
      </div>
      <button
        className="selectpill"
        type="button"
        onClick={() => (active.locked ? lockedToast() : onSelect(active))}
      >
        انتخاب {active.name}
      </button>
    </div>
  );
}

/* ---------- guide detail ---------- */
export function GuideDetail({ guide, onClose, onSelect }) {
  const bars = useRef(
    Array.from({ length: 46 }, (_, i) => 6 + Math.abs(Math.sin(i * 1.7)) * 30),
  );
  return (
    <div className="screen gdetail pop">
      <div className="hero">
        <img src={guide.img} alt={guide.name} />
        <div className="fade" />
        <button className="close" type="button" onClick={onClose} aria-label="بستن"><X /></button>
        <div className="meta">
          <div className="eyebrow-w">راهنمای تو</div>
          <h2 className="name">{guide.name}</h2>
          <div className="tags">
            {guide.tags.map((t) => <span className="gtag" key={t}>{t}</span>)}
          </div>
        </div>
      </div>
      <div className="body">
        <div className="listenrow">
          <button className="listen" type="button"><span>گوش کن</span> <Waveform size={19} /></button>
          <div className="wave">
            {bars.current.map((h, i) => <i key={i} style={{ height: h }} />)}
          </div>
        </div>
        <p className="quote">{guide.quote}</p>
        <hr />
        <h4>کمی دربارهٔ من</h4>
        {guide.about.map((p, i) => <p className="p" key={i}>{p}</p>)}
        <hr />
        <h4>این‌طور پیش می‌رویم</h4>
        {guide.how.map((p, i) => <p className="p" key={i}>{p}</p>)}
      </div>
      <div className="selectbar">
        <Pill variant="light" onClick={() => onSelect(guide)} disabled={guide.locked}>
          انتخاب {guide.name}
        </Pill>
      </div>
    </div>
  );
}

/* ---------- expressions permission ---------- */
export function Expressions({ onClose, onAccept, onDecline }) {
  return (
    <div className="screen expr pop">
      <div className="scroll" style={{ padding: '18px 0 8px' }}>
        <div style={{ display: 'flex', justifyContent: 'flex-end', padding: '0 22px' }}>
          <button type="button" style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--tx)' }} onClick={onClose} aria-label="بستن">
            <X />
          </button>
        </div>
        <div className="illus">
          <svg width="100%" height="100%" viewBox="0 0 342 210" fill="none" aria-hidden>
            <circle cx="171" cy="105" r="52" fill="#bfe0e4" />
            <circle cx="171" cy="105" r="86" stroke="#a9cfd4" strokeWidth="1.4" />
            <circle cx="171" cy="105" r="120" stroke="#a9cfd4" strokeWidth="1.4" />
            <circle cx="171" cy="105" r="38" fill="#4d8f96" />
            <g transform="translate(156 90)" stroke="#eaf6f7" strokeWidth="2">
              <path d="M2 15s5-9 13-9 13 9 13 9-5 9-13 9S2 15 2 15Z" />
              <circle cx="15" cy="15" r="4" />
            </g>
            <circle cx="278" cy="48" r="24" fill="#fff" />
            <g transform="translate(267 37)" stroke="#4d8f96" strokeWidth="1.8">
              <path d="M3 8h3l1.6-2.4h6.8L16 8h3a1.4 1.4 0 0 1 1.4 1.4v8A1.4 1.4 0 0 1 19 18.8H4a1.4 1.4 0 0 1-1.4-1.4v-8A1.4 1.4 0 0 1 4 8Z" />
              <circle cx="11" cy="13" r="3.2" />
            </g>
            <circle cx="262" cy="150" r="24" fill="#fff" />
            <g transform="translate(251 139)" stroke="#4d8f96" strokeWidth="1.8">
              <rect x="2.8" y="6.4" width="12.6" height="11.2" rx="2.4" />
              <path d="m15.4 10.6 5.8-3v8.8l-5.8-3" />
            </g>
          </svg>
        </div>
        <div style={{ padding: '34px 24px 0' }}>
          <p className="eyebrow teal">حالت چهره</p>
          <h1 className="h-serif expr-h">جنس تازه‌ای<br />از هم‌نوایی</h1>
          <p className="lede">
            رهراه می‌تواند سیگنال‌هایی از حالت چهرهٔ تو بگیرد؛ این به ما کمک می‌کند بفهمیم بیرون از واژه‌ها چه احساسی داری.
          </p>
          <div className="datacard">
            <h5>با داده‌ات چه می‌کنیم</h5>
            <div className="line">
              <CheckCircle size={24} />
              <span>فقط برای بهتر کردن تجربهٔ درمانت استفاده می‌شود. هرگز به اشتراک گذاشته یا فروخته نمی‌شود.</span>
            </div>
          </div>
          <p className="fine">
            <b>ما سیگنال می‌گیریم، نه قطعیت.</b> ممکن است اشتباه کنیم، و اگر چنین شود خودمان به تو می‌گوییم.
          </p>
        </div>
      </div>
      <div className="btn-row gap">
        <Pill variant="dark" onClick={onAccept}>روشن کردن حالت چهره</Pill>
        <Pill variant="outline" onClick={onDecline}>الان نه</Pill>
      </div>
    </div>
  );
}

/* ---------- session duration ---------- */
export function Duration({ onBack, onStart }) {
  const vals = [10, 15, 20, 25, 30];
  const [sel, setSel] = useState(20);
  return (
    <div className="screen dur">
      <div className="topbar on-dark">
        <button className="iconbtn" onClick={onBack} aria-label="بازگشت" type="button"><Back /></button>
        <span className="brand" style={{ fontSize: 26 }}>مدت جلسه</span>
      </div>
      <div className="wheel">
        {vals.map((v, i) => {
          const d = Math.abs(v - sel) / 5;
          return (
            <button
              key={v}
              type="button"
              className="slot"
              onClick={() => setSel(v)}
              style={{
                fontSize: v === sel ? 42 : d === 1 ? 30 : 24,
                opacity: v === sel ? 1 : d === 1 ? 0.55 : 0.3,
              }}
            >
              {faNum(v)} دقیقه
            </button>
          );
        })}
      </div>
      <p className="cap2">هر وقت بخواهی می‌توانی جلسه را تمدید کنی یا زودتر ترک کنی.</p>
      <div className="btn-row">
        <Pill variant="light" onClick={() => onStart(sel)}>شروع جلسه</Pill>
      </div>
    </div>
  );
}
