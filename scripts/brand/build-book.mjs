import { SCENES, PALETTES, companion, cairn, vessel } from '../../src/brand/scenes.js';
import { pebblePath } from '../../src/brand/geometry.js';
import fs from 'node:fs';
import { join } from 'node:path';

const root = join(import.meta.dirname, '..', '..');
const liveBook = join(root, 'public/brand/Hamrah-Brand-Book.html');
const archivedBook = join(root, 'docs/brand/Hamrah-Brand-Book.html');

// The book is set in the same three faces the app already ships (@fontsource-variable).
// They are copied next to the book so it loads offline and never drifts from the product.
const FONT_SOURCES = [
  ['@fontsource-variable/estedad/files/estedad-arabic-wght-normal.woff2', 'estedad-arabic.woff2'],
  ['@fontsource-variable/estedad/files/estedad-latin-wght-normal.woff2', 'estedad-latin.woff2'],
  ['@fontsource-variable/noto-naskh-arabic/files/noto-naskh-arabic-arabic-wght-normal.woff2', 'naskh-arabic.woff2'],
];
const fontDir = join(root, 'public/brand/fonts');
fs.mkdirSync(fontDir, { recursive: true });
for (const [from, to] of FONT_SOURCES) {
  const source = join(root, 'node_modules', from);
  if (!fs.existsSync(source)) {
    throw new Error(`brand book: ${from} is missing — run npm install first`);
  }
  fs.copyFileSync(source, join(fontDir, to));
}

/* ---------- helpers ---------- */
const layer = (l) => l.t === 'circle'
  ? `<circle cx="${l.cx}" cy="${l.cy}" r="${l.r}" fill="${l.fill}"/>`
  : `<path d="${l.d}" fill="${l.fill}"${l.stroke ? ` stroke="${l.stroke}" stroke-width="${l.width}" stroke-linecap="${l.cap}" stroke-linejoin="round"` : ''}${l.rule ? ` fill-rule="${l.rule}"` : ''}${l.dash ? ` stroke-dasharray="${l.dash}"` : ''}/>`;
const sceneSvg = (name, pal, attrs = '') => {
  const sc = SCENES[name](PALETTES[pal]);
  return `<svg ${attrs} viewBox="${sc.viewBox}" preserveAspectRatio="xMidYMid meet" role="img" aria-hidden="true">${sc.layers.map(layer).join('')}</svg>`;
};
const raw = (layers, vb, attrs='') => `<svg ${attrs} viewBox="${vb}" preserveAspectRatio="xMidYMid meet" aria-hidden="true">${layers.map(layer).join('')}</svg>`;

/* ---------- page frame ---------- */
const CSS = `
  @font-face{font-family:Estedad;src:url(fonts/estedad-arabic.woff2) format('woff2-variations');font-weight:100 900;}
  @font-face{font-family:EstedadLatin;src:url(fonts/estedad-latin.woff2) format('woff2-variations');font-weight:100 900;}
  @font-face{font-family:Naskh;src:url(fonts/naskh-arabic.woff2) format('woff2-variations');font-weight:400 700;}
  :root{
    --ink:#23201b; --dark:#12110d; --marble:#f3f2ee; --sand:#cec4b8; --ember:#c1664c;
    --mist:#d5d9f7; --sun:#f6cf98; --moss:#9eda9e; --clay:#c9c1b4; --muted:#6f6a61;
    --rule:#ddd7cc; --paper:#faf9f6;
  }
  *{box-sizing:border-box;}
  html{scroll-behavior:smooth;}
  body{margin:0;background:var(--paper);color:var(--ink);
    font-family:Estedad,EstedadLatin,Tahoma,sans-serif;direction:rtl;
    -webkit-font-smoothing:antialiased;line-height:1.85;}
  .page{max-width:1120px;margin:0 auto;padding:0 40px 120px;}
  section{padding:88px 0 24px;border-top:1px solid var(--rule);}
  section:first-of-type{border-top:0;}
  .eyebrow{font-size:12px;letter-spacing:.16em;color:var(--muted);text-transform:uppercase;
    font-family:EstedadLatin,Estedad;margin:0 0 18px;direction:ltr;text-align:right;}
  h1{font-family:Naskh,Estedad;font-weight:500;font-size:64px;line-height:1.15;margin:0 0 18px;letter-spacing:-.01em;}
  h2{font-size:30px;font-weight:700;margin:0 0 10px;line-height:1.35;}
  h3{font-size:19px;font-weight:650;margin:0 0 8px;}
  p{margin:0 0 14px;font-size:16px;color:#3b3730;}
  .lead{font-size:20px;line-height:1.85;color:#3b3730;max-width:62ch;}
  .small{font-size:13px;color:var(--muted);}
  .mono{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;direction:ltr;text-align:left;font-size:12.5px;}
  .grid{display:grid;gap:22px;}
  .g2{grid-template-columns:repeat(2,minmax(0,1fr));}
  .g3{grid-template-columns:repeat(3,minmax(0,1fr));}
  .g4{grid-template-columns:repeat(4,minmax(0,1fr));}
  .card{background:#fff;border:1px solid var(--rule);border-radius:20px;padding:26px;}
  .swatch{border-radius:18px;overflow:hidden;border:1px solid rgb(0 0 0 / 6%);}
  .swatch .chip{height:132px;}
  .swatch .meta{padding:14px 16px;background:#fff;}
  .swatch b{display:block;font-size:15px;}
  .swatch code{font-family:ui-monospace,monospace;font-size:12px;color:var(--muted);direction:ltr;}
  .hero{min-height:520px;display:grid;grid-template-columns:1.1fr .9fr;gap:40px;align-items:center;padding-top:64px;}
  .heroArt{background:var(--marble);border-radius:32px;padding:26px;}
  .heroArt.dark{background:var(--dark);}
  .lockupDemo{display:flex;align-items:center;gap:18px;}
  .wm{font-family:Naskh,Estedad;font-size:74px;line-height:1;font-weight:500;}
  .tagline{font-size:19px;color:var(--muted);}
  .icons{display:flex;align-items:flex-end;gap:26px;flex-wrap:wrap;}
  .icons figure{margin:0;text-align:center;}
  .icons svg{display:block;margin:0 auto 8px;color:var(--ink);}
  .icons figcaption{font-size:11px;color:var(--muted);font-family:ui-monospace,monospace;}
  .tileRow{display:flex;gap:18px;flex-wrap:wrap;align-items:center;}
  .tile{border-radius:28%;display:grid;place-items:center;background:var(--dark);}
  .tile.rev{background:var(--marble);}
  .tile svg{width:64%;height:64%;}
  .screen{background:#fff;border:1px solid var(--rule);border-radius:22px;overflow:hidden;}
  .screen .frame{position:relative;aspect-ratio:430/720;background:var(--marble);overflow:hidden;}
  .screen .frame.dark{background:var(--dark);}
  .screen .frame.beige{background:var(--sand);}
  .screen svg.art{position:absolute;inset:auto;}
  .screen .caption{padding:12px 16px;font-size:12.5px;color:var(--muted);border-top:1px solid var(--rule);}
  .ttl{font-family:Estedad;font-weight:730;font-size:22px;line-height:1.4;}
  .ttl.dark{color:#f4f3ef;}
  .sub{font-size:13.5px;color:#494439;line-height:1.75;}
  .sub.dark{color:#b9b5ad;}
  .btn{position:absolute;left:16px;right:16px;bottom:14px;height:46px;border-radius:15px;
    background:var(--ink);color:#fff;display:grid;place-items:center;font-size:14px;font-weight:600;}
  .btn.light{background:#fff;color:#171713;}
  .head{position:absolute;top:34px;right:20px;left:20px;}
  .rule{height:1px;background:var(--rule);margin:26px 0;}
  .anatomy{position:relative;background:var(--marble);border-radius:22px;padding:24px;}
  .notes{display:grid;gap:14px;margin-top:18px;}
  .note{display:grid;grid-template-columns:26px 1fr;gap:12px;align-items:start;font-size:14.5px;color:#3b3730;}
  .note i{width:26px;height:26px;border-radius:50%;background:var(--ink);color:#fff;display:grid;place-items:center;
    font-size:12px;font-style:normal;font-family:EstedadLatin;}
  .dont{background:#fbeae5;border-color:#efcabc;}
  .do{background:#eaf4ea;border-color:#c9e3c9;}
  .do h3,.dont h3{font-size:15px;}
  .quote{font-family:Naskh;font-size:26px;line-height:1.9;border-right:3px solid var(--ember);padding-right:20px;margin:26px 0;}
  table{width:100%;border-collapse:collapse;font-size:14px;}
  th,td{text-align:right;padding:12px 10px;border-bottom:1px solid var(--rule);vertical-align:top;}
  th{font-size:12px;color:var(--muted);font-weight:600;letter-spacing:.04em;}
  .pill{display:inline-block;padding:2px 10px;border-radius:99px;background:#eee9df;font-size:11.5px;color:#5b554a;}
  .motion{display:flex;gap:12px;align-items:center;}
  @keyframes ember{0%,100%{opacity:1;transform:scale(1)}50%{opacity:.72;transform:scale(.9)}}\n  .kbd{font-family:ui-monospace,monospace;background:#eae5db;border-radius:6px;padding:1px 6px;font-size:12px;}
`;

const head = `<!doctype html><html lang="fa" dir="rtl"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>همراه — کتاب برند</title><style>${CSS}</style></head><body><main class="page">`;
const foot = `</main></body></html>`;

/* ---------- 01 cover ---------- */
const cover = `
<section class="hero">
  <div>
    <p class="eyebrow">Hamrah — visual identity</p>
    <h1>سنگِ همراه</h1>
    <p class="lead">هویت بصری همراه از یک شکل ساده ساخته شده: <b>دو سنگ که به هم تکیه می‌دهند</b> و
    کورسویی که در شکاف میان‌شان روشن است. همین یک شکل، هم نشان همراه است و هم شخصیت‌هایش.</p>
    <p class="small">نسخهٔ ۱٫۰ — ۱۲ آبان ۱۴۰۴</p>
  </div>
  <div class="heroArt">${sceneSvg('mark', 'light', 'style="width:100%;height:auto;display:block"')}</div>
</section>`;

/* ---------- 02 idea ---------- */
const idea = `
<section>
  <p class="eyebrow">01 — the idea</p>
  <h2>هم‌سنگ: سنگی که با تو می‌نشیند</h2>
  <p class="lead">در فارسی، «همراه» کسی است که با تو می‌آید. سنگ هم صبورترین چیز جهان است:
  نه می‌شتابد، نه قضاوت می‌کند، و سال‌ها همان‌جا می‌ماند. در کوهنوردی هم سنگ‌چین می‌گذارند تا
  مسیر را نشان دهد و بگوید «یکی از این‌جا گذشته؛ تو هم می‌توانی.»</p>
  <div class="grid g3" style="margin-top:26px">
    <div class="card"><h3>دو، نه یکی</h3><p class="small" style="margin:0">همراه یعنی دیگری. نشان از دو سنگ ساخته شده که هیچ‌کدام به‌تنهایی کامل نیست.</p></div>
    <div class="card"><h3>شکاف، نه نقطه</h3><p class="small" style="margin:0">جایی که دو سنگ به هم می‌رسند خالی نمی‌ماند؛ کورسویی همان‌جاست. توجه، همان نور است.</p></div>
    <div class="card"><h3>سکون، نه سکوت</h3><p class="small" style="margin:0">سنگ‌ها آرام‌اند، اما بی‌حرکت نه: می‌نشینند، تکیه می‌دهند، پله می‌شوند و در مسیر می‌مانند.</p></div>
  </div>
  <p class="quote">همراه، سنگی است که کنارت می‌نشیند — نه سنگی که روی دلت می‌ماند.</p>
</section>`;

/* ---------- 03 mark anatomy ---------- */
const anatomy = `
<section>
  <p class="eyebrow">02 — the mark</p>
  <h2>ساختار نشان</h2>
  <p class="lead">سه عنصر، سه قاعده. هر چیزی که به این هویت اضافه شود باید از همین سه قاعده بیرون بیاید.</p>
  <div class="grid g2" style="margin-top:24px;align-items:start">
    <div class="anatomy">
      <svg viewBox="0 0 300 300" style="width:100%;height:auto" aria-hidden="true">
        <rect width="300" height="300" fill="none"/>
        <g transform="translate(30,30)">${SCENES.mark(PALETTES.light).layers.map(layer).join('')}</g>
        <g stroke="#c1664c" stroke-width="1.2" fill="none" opacity=".85">
          <circle cx="160" cy="134" r="11"/>
          <path d="M160 60 V110" stroke-dasharray="3 3"/>
          <path d="M226 60 V90" stroke-dasharray="3 3"/>
          <path d="M92 240 H262" stroke-dasharray="3 3"/>
        </g>
        <circle cx="160" cy="134" r="2.4" fill="#c1664c"/>
      </svg>
      <div class="notes">
        <div class="note"><i>1</i><div><b>دو سنگ.</b> کوچک‌تر به بزرگ‌تر تکیه می‌دهد. هیچ‌وقت هم‌اندازه نیستند؛ نابرابری، همان رابطه است.</div></div>
        <div class="note"><i>2</i><div><b>کورسو.</b> یک دایرهٔ گرم در شکاف، کمی بالاتر از مرکز بصری. تنها رنگ گرمِ نشان.</div></div>
        <div class="note"><i>3</i><div><b>خط زمین.</b> هر دو سنگ روی یک خط می‌نشینند؛ در نسخهٔ نشان حذف می‌شود، در صحنه‌ها می‌ماند.</div></div>
      </div>
    </div>
    <div>
      <div class="card" style="margin-bottom:18px">
        <h3>تناسب</h3>
        <p class="small" style="margin:0">سنگ بلند ۲ برابر سنگ کوتاه است؛ عرض‌ها ۵۲ و ۴۶. فاصلهٔ مرکزها ۸۲ از ۲۴۰.
        کورسو دقیقاً در برخورد دو منحنی می‌نشیند، نه در مرکز هندسی.</p>
      </div>
      <div class="card" style="margin-bottom:18px">
        <h3>حداقل اندازه</h3>
        <p class="small" style="margin:0">۱۶ پیکسل برای نمایش دیجیتال. زیر ۱۶ پیکسل فقط نسخهٔ تک‌رنگ روی زمینهٔ تیره مجاز است.</p>
      </div>
      <div class="card">
        <h3>فضای آزاد</h3>
        <p class="small" style="margin:0">حداقل به اندازهٔ قطر کورسو (۹ واحد) از هر طرف. هیچ متنی داخل این حاشیه نمی‌نشیند.</p>
      </div>
    </div>
  </div>
</section>`;

/* ---------- 04 sizes ---------- */
const sizesRow = [16,20,24,32,48,64,96,144].map(s =>
  `<figure><svg width="${s}" height="${s}" viewBox="0 0 240 240" aria-hidden="true">${SCENES.mark(PALETTES.light).layers.map(layer).join('')}</svg><figcaption>${s}px</figcaption></figure>`
).join('');
const sizesRowRev = [16,20,24,32,48,64,96,144].map(s =>
  `<figure><svg width="${s}" height="${s}" viewBox="0 0 240 240" aria-hidden="true">${SCENES.mark({...PALETTES.light, stone:'#f3f2ee'}).layers.map(layer).join('')}</svg><figcaption>${s}px</figcaption></figure>`
).join('');
const sizes = `
<section>
  <p class="eyebrow">03 — scale & polarity</p>
  <h2>در هر اندازه، همان نشان</h2>
  <p class="lead">نشان تا ۱۶ پیکسل هم خواناست، چون فقط دو توده و یک نقطه دارد. پلاریته‌ها ثابت‌اند: جوهر روی مرمر، مرمر روی جوهر، و تک‌رنگ برای چاپ و حکاکی.</p>
  <div class="card" style="margin-top:22px"><div class="icons">${sizesRow}</div></div>
  <div class="card" style="margin-top:18px;background:var(--dark);border-color:#2a2822"><div class="icons" style="color:#f3f2ee">${sizesRowRev}</div></div>
  <div class="tileRow" style="margin-top:22px">
    <span class="tile" style="width:88px;height:88px"><svg viewBox="0 0 240 240" aria-hidden="true">${SCENES.mark({...PALETTES.light, stone:'#f3f2ee'}).layers.map(layer).join('')}</svg></span>
    <span class="tile rev" style="width:88px;height:88px"><svg viewBox="0 0 240 240" aria-hidden="true">${SCENES.mark(PALETTES.light).layers.map(layer).join('')}</svg></span>
    <span class="tile" style="width:88px;height:88px;background:var(--ember)"><svg viewBox="0 0 240 240" aria-hidden="true">${SCENES.mark({...PALETTES.light, stone:'#f3f2ee', ember:'#12110d'}).layers.map(layer).join('')}</svg></span>
    <span class="tile" style="width:88px;height:88px;background:var(--mist)"><svg viewBox="0 0 240 240" aria-hidden="true">${SCENES.mark(PALETTES.light).layers.map(layer).join('')}</svg></span>
  </div>
</section>`;

/* ---------- 05 lockup ---------- */
const lockup = `
<section>
  <p class="eyebrow">04 — lockup</p>
  <h2>نشان و نام</h2>
  <p class="lead">نام همراه با خط نسخِ خودِ محصول نوشته می‌شود؛ نشان در سمت راست (شروعِ خواندن) می‌نشیند و
  شعار در دو کلمه زیر آن می‌آید. فاصلهٔ ثابت: ۱۲ واحد، برابر با فاصلهٔ دو سنگ در نشان.</p>
  <div class="grid g2" style="margin-top:24px">
    <div class="card"><div class="lockupDemo"><span style="width:64px;height:64px;display:block">${sceneSvg('mark','light','style="width:64px;height:64px"')}</span>
      <span><span class="wm">همراه</span><br><span class="tagline">با هم، آرام‌تر</span></span></div></div>
    <div class="card" style="background:var(--dark);border-color:#2a2822"><div class="lockupDemo"><span style="width:64px;height:64px;display:block">${sceneSvg('mark','light','style="width:64px;height:64px"')}</span>
      <span><span class="wm" style="color:#f3f2ee">همراه</span><br><span class="tagline" style="color:#a8a29a">با هم، آرام‌تر</span></span></div></div>
  </div>
  <div class="grid g3" style="margin-top:18px">
    <div class="card do"><h3>درست</h3><p class="small" style="margin:0">شعار فقط وقتی جا هست. نشان کوچک‌تر از نام، هرگز بزرگ‌تر.</p></div>
    <div class="card dont"><h3>نادرست</h3><p class="small" style="margin:0">شکستن نشان به دو آیکون جدا، چرخاندن آن، یا افزودن سایه و گرادیان.</p></div>
    <div class="card dont"><h3>نادرست</h3><p class="small" style="margin:0">کشیدن نشان روی عکس، یا استفاده از کورسو به‌عنوان رنگ متن.</p></div>
  </div>
</section>`;

/* ---------- 06 color ---------- */
const colors = [
  ['جوهر', '#23201b', 'متن، سنگ‌ها، دکمهٔ اصلی', 'ink'],
  ['شب', '#12110d', 'صفحه‌های خانه و گفت‌وگو', 'dark'],
  ['مرمر', '#f3f2ee', 'پس‌زمینهٔ روشن', 'marble'],
  ['کورسو', '#c1664c', 'تنها رنگ گرم؛ نقطهٔ توجه', 'ember'],
  ['ماسه', '#cec4b8', 'لایهٔ میانی، صفحه‌های آرام', 'sand'],
  ['مه', '#d5d9f7', 'کنش‌های آرام، حباب گفت‌وگو', 'mist'],
  ['آفتاب', '#f6cf98', 'کارت تأمل و نامه', 'sun'],
  ['خزه', '#9eda9e', 'بینش، وضعیت سالم', 'moss'],
];
const color = `
<section>
  <p class="eyebrow">05 — colour</p>
  <h2>یک رنگ گرم، بقیه سکوت</h2>
  <p class="lead">پالت از خود محصول آمده و دست‌نخورده مانده است. قانون تازه فقط یکی است:
  <b>در هر قاب، تنها یک چیز می‌تواند کورسویی باشد.</b> اگر دو نقطهٔ گرم دیدی، یکی‌شان اضافه است.</p>
  <div class="grid g4" style="margin-top:24px">
    ${colors.map(([n,h,u,k]) => `<div class="swatch"><div class="chip" style="background:${h};${k==='marble'||k==='sun'||k==='mist'||k==='moss'?'box-shadow:inset 0 0 0 1px #0001':''}"></div>
      <div class="meta"><b>${n}</b><code>${h.toUpperCase()}</code><p class="small" style="margin:4px 0 0">${u}</p></div></div>`).join('')}
  </div>
</section>`;

/* ---------- 07 characters ---------- */
const charRow = (() => {
  const poses = [
    ['stand', 'ایستاده'], ['walk', 'در راه'], ['wave', 'سلام'], ['offer', 'پیشکش'],
  ];
  return poses.map(([p, n]) => {
    const layers = companion({ x: 110, base: 250, scale: 1.25, pose: p, ember: false, palette: PALETTES.light });
    return `<div class="card" style="padding:14px"><svg viewBox="0 0 220 280" style="width:100%;height:auto" aria-hidden="true">
      ${layers.map(layer).join('')}</svg><p class="small" style="text-align:center;margin:6px 0 0">${n}</p></div>`;
  }).join('');
})();
const charDarkRow = (() => {
  const sc = [
    companion({ x: 96, base: 250, scale: 1.1, pose: 'stand', lean: 10, ember: false, palette: PALETTES.dark }),
    companion({ x: 96, base: 250, scale: 1.1, pose: 'sit', lean: -6, ember: false, palette: PALETTES.dark, flip: true }),
  ];
  const c = cairn({ x: 150, base: 258, scale: 1.2, count: 4, palette: PALETTES.dark });
  return `<div class="card" style="background:var(--dark);border-color:#2a2822">
    <div class="grid g3" style="gap:14px">
      <svg viewBox="0 0 200 290" style="width:100%" aria-hidden="true">${sc[0].map(layer).join('')}</svg>
      <svg viewBox="0 0 200 290" style="width:100%" aria-hidden="true">${sc[1].map(layer).join('')}</svg>
      <svg viewBox="0 0 300 290" style="width:100%" aria-hidden="true">${c.map(layer).join('')}</svg>
    </div>
    <p class="small" style="color:#a8a29a;text-align:center;margin:8px 0 0">ایستاده · در حال استراحت · سنگ‌چین</p>
  </div>`;
})();
const characters = `
<section>
  <p class="eyebrow">06 — characters</p>
  <h2>سنگ‌ها راه می‌روند</h2>
  <p class="lead">شخصیت‌های همراه همان سنگ‌های نشان‌اند: بدنی از جنس جوهر، کورسویی در سینه، و دو پای کوچک.
  <b>هیچ چهره‌ای ندارند.</b> حالت را قامت می‌گوید، نه چشم و لب — دقیقاً همان اندازه‌ای که یک اپ سلامت روان باید ساکت بماند.</p>
  <div class="grid g4" style="margin-top:24px">${charRow}</div>
  <div style="margin-top:18px">${charDarkRow}</div>
  <div class="grid g3" style="margin-top:18px">
    <div class="card"><h3>قامت، حالت را می‌گوید</h3><p class="small" style="margin:0">کج‌شدن به سمت دیگری یعنی توجه؛ فاصلهٔ پاها یعنی شتاب؛ کوتاه‌شدن یعنی استراحت.</p></div>
    <div class="card"><h3>چهره نداریم</h3><p class="small" style="margin:0">هیچ چشم، دهان یا اشکی. این محدودیت، از ادعای درمان و از سوگیری احساسی دور می‌ماند.</p></div>
    <div class="card"><h3>کورسو، همیشه در سینه</h3><p class="small" style="margin:0">کورسو فقط وقتی خاموش می‌شود که شخصیت در حال برقراری ارتباط با شخصیت دیگری باشد.</p></div>
  </div>
</section>`;


/* ---------- 07b. signature detail ---------- */
const signature = `
<section>
  <p class="eyebrow">06b — the signature</p>
  <h2>کورسویی که نفس می‌کشد</h2>
  <p class="lead">در کل محصول فقط یک چیز گرم است، پس فقط یک چیز حق حرکت دارد: کورسوی نشان.
  وقتی همراه گوش می‌دهد آرام (۴ ثانیه) و وقتی فکر می‌کند تندتر (۱٫۶ ثانیه) نفس می‌کشد.
  بقیهٔ صفحه‌ها ساکن‌اند. در حالت کاهش حرکت، کورسو فقط روشن می‌ماند.</p>
  <div class="grid g3" style="margin-top:22px">
    <div class="card" style="text-align:center">
      <svg width="64" height="64" viewBox="0 0 240 240" aria-hidden="true" style="animation:ember 4s ease-in-out infinite">${SCENES.mark(PALETTES.light).layers.map(layer).join('')}</svg>
      <p class="small" style="margin:8px 0 0">در حال گوش‌دادن · ۴s</p>
    </div>
    <div class="card" style="text-align:center;background:var(--dark);border-color:#2a2822">
      <svg width="64" height="64" viewBox="0 0 240 240" aria-hidden="true" style="animation:ember 1.6s ease-in-out infinite">${SCENES.mark({...PALETTES.light, stone:'#f3f2ee'}).layers.map(layer).join('')}</svg>
      <p class="small" style="margin:8px 0 0;color:#a8a29a">در حال فکرکردن · ۱٫۶s</p>
    </div>
    <div class="card" style="text-align:center">
      <svg width="64" height="64" viewBox="0 0 240 240" aria-hidden="true">${SCENES.mark(PALETTES.light).layers.map(layer).join('')}</svg>
      <p class="small" style="margin:8px 0 0">در حالت سکون · ثابت</p>
    </div>
  </div>
</section>`;

/* ---------- 07c. character pose sheet ---------- */
const poseSheetSection = () => `
<section>
  <p class="eyebrow">06c — the cast</p>
  <h2>هفت حالت، یک سنگ</h2>
  <p class="lead">همهٔ شخصیت‌ها از یک سنگ ساخته می‌شوند. تفاوت فقط در قامت است؛ به همین دلیل خانواده‌شان در هر ترکیبی بازشناختنی می‌ماند.</p>
  <div class="grid" style="grid-template-columns:repeat(7,minmax(0,1fr));gap:12px;margin-top:22px">${poseSheet}</div>
</section>`;

const poseSheet = (() => {
  const poses = ['stand','walk','wave','offer','reach','hold','sit'];
  const labels = ['ایستاده','در راه','سلام','پیشکش','دست‌درازکردن','تکیه','استراحت'];
  return poses.map((p, i) => {
    const layers = companion({ x: 105, base: 250, scale: 1.15, pose: p, ember: true, palette: PALETTES.light });
    return `<figure style="margin:0;text-align:center">
      <div class="card" style="padding:10px;background:var(--marble)">
        <svg viewBox="0 0 210 280" style="width:100%;height:auto" aria-hidden="true">${layers.map(layer).join('')}</svg>
      </div>
      <figcaption class="small" style="margin-top:6px">${labels[i]}<br><span class="mono" style="font-size:10.5px">${p}</span></figcaption>
    </figure>`;
  }).join('');
})();

/* ---------- 08 scenes / screens ---------- */
const screenCell = (title, sub, art, opts = {}) => {
  const { dark = false, beige = false, btn = 'ادامه', light = false, caption = '' } = opts;
  return `<figure class="screen" style="margin:0">
    <div class="frame ${dark ? 'dark' : beige ? 'beige' : ''}">
      <div class="head">
        <div class="ttl ${dark ? 'dark' : ''}">${title}</div>
        ${sub ? `<div class="sub ${dark ? 'dark' : ''}" style="margin-top:8px">${sub}</div>` : ''}
      </div>
      <div style="position:absolute;top:150px;left:14px;right:14px;bottom:78px;display:grid;place-items:center">${art}</div>
      <div class="btn ${light ? 'light' : ''}">${btn}</div>
    </div>
    <figcaption class="caption">${caption}</figcaption>
  </figure>`;
};
const scenes = `
<section>
  <p class="eyebrow">07 — illustration</p>
  <h2>یک صحنه، یک جمله</h2>
  <p class="lead">هر تصویر فقط یک اتفاق را نشان می‌دهد: اولین سنگ، پله‌شدن، تکیه‌دادن، یا کورسویی که می‌رسد.
  زمینه همیشه ساده است و خط زمین، همه را به هم وصل می‌کند.</p>
  <div class="grid g3" style="margin-top:24px">
    ${screenCell('با همراه سلام کن.', 'همراه، هوشی مصنوعی برای سلامت روان و حمایت عاطفی است.', sceneSvg('welcome','light','style="width:100%;height:auto"'), { btn: 'شروع کنیم', caption: 'خوش‌آمد — دو سنگ و یک رشتهٔ توجه' })}
    ${screenCell('حالا کمی بهتر بشناسیمت.', 'به همراه کمک کن تصویر بزرگ‌تری ببیند.', sceneSvg('know-intro','dark','style="width:100%;height:auto"'), { dark: true, light: true, btn: 'آماده‌ام', caption: 'میان‌صفحه — سنگ‌ها، حافظهٔ مسیر' })}
    ${screenCell('همراهِ تو بالاخره آماده است.', 'نام و سبک گفت‌وگویت ثبت می‌شود.', sceneSvg('ready','light','style="width:100%;height:auto"'), { btn: 'آماده‌ام', caption: 'آماده — با هم راه می‌افتند' })}
    ${screenCell('تعهد ما به تو.', 'گفت‌وگوهای تو برای ادامهٔ گفتگو در حساب همراه نگهداری می‌شوند.', `<div style="width:78%">${sceneSvg('privacy','light','style="width:100%;height:auto"')}</div>`, { btn: 'ادامه', caption: 'حریم خصوصی — سنگِ ظرف، کورسو را نگه می‌دارد' })}
    ${screenCell('حالا کمی عمیق‌تر برویم.', 'به چند پرسش کوتاه جواب بده تا همراه بفهمد چرا اینجایی.', sceneSvg('deeper-intro','dark','style="width:100%;height:auto"'), { dark: true, light: true, btn: 'آماده‌ام', caption: 'میان‌صفحه — سنگ‌ها پله می‌شوند' })}
    ${screenCell('اسمت چیست؟', 'همراه با همین اسم صدایت می‌کند.', sceneSvg('name','light','style="width:100%;height:auto"'), { btn: 'ادامه', caption: 'نام — کورسو، بالای سرِ سنگ' })}
  </div>
</section>`;

/* ---------- 09 in-product moments ---------- */
const product = `
<section>
  <p class="eyebrow">08 — in product</p>
  <h2>امضای همراه در خودِ محصول</h2>
  <p class="lead">نشان در جاهایی می‌نشیند که کاربر با آن زندگی می‌کند: مُهر نامهٔ خانه، منوی کنار،
  سرصفحهٔ دفتر گفت‌وگو، و صفحهٔ ورود.</p>
  <div class="grid g3" style="margin-top:24px">
    <div class="card" style="background:var(--dark);border-color:#2a2822;padding:22px">
      <p class="small" style="color:#a8a29a;margin:0 0 14px">مُهرِ نامهٔ روزانه</p>
      <div style="position:relative;width:200px;height:120px;border:1px solid #3b3936;border-radius:12px;background:#252422;transform:rotate(2deg);margin:0 auto">
        <div style="position:absolute;left:50%;top:62%;transform:translate(-50%,-50%);width:54px;height:54px;border-radius:50%;background:#b95f4f;box-shadow:inset 0 0 0 4px #ca7260, inset 0 0 0 6px #9d4e42;display:grid;place-items:center">
          ${sceneSvg('mark','light','style="width:30px;height:30px;display:block;filter:brightness(3.6) saturate(0.45)"')}
        </div>
      </div>
      <p class="small" style="color:#a8a29a;margin:14px 0 0">نشان، داخل مُهر موم — تنها جای برجستهٔ نامه.</p>
    </div>
    <div class="card" style="padding:22px">
      <p class="small" style="color:var(--muted);margin:0 0 14px">منوی کنار</p>
      <div style="display:flex;align-items:center;gap:12px;padding:16px;background:var(--dark);border-radius:16px;color:#e8e6e1">
        ${sceneSvg('mark','light','style="width:34px;height:34px;display:block;filter:brightness(3.4)"')}
        <span style="font-family:Naskh,Estedad;font-size:30px">همراه</span>
      </div>
      <p class="small" style="margin:14px 0 0">نشان پیش از نام می‌نشیند؛ نام با خط نسخ، همان‌طور که در محصول هست.</p>
    </div>
    <div class="card" style="padding:22px">
      <p class="small" style="color:var(--muted);margin:0 0 14px">صفحهٔ ورود</p>
      <div style="text-align:center;padding:18px 0">
        ${sceneSvg('mark','light','style="width:46px;height:46px;display:block;margin:0 auto 6px"')}
        <div style="font-family:Naskh,Estedad;font-size:26px">همراه</div>
        <div class="small">با هم، آرام‌تر</div>
      </div>
      <p class="small" style="margin:0">قفل‌واژهٔ برند: نشان، نام، شعار — به همین ترتیب.</p>
    </div>
  </div>
</section>`;

/* ---------- 10 typography ---------- */
const type = `
<section>
  <p class="eyebrow">09 — typography</p>
  <h2>خط، همان خطِ محصول</h2>
  <p class="lead">دو خط موجود دست‌نخورده می‌مانند. استداد برای همهٔ رابط؛ نسخ برای نام برند و جمله‌های
  آرام (سلام خانه، مُهر، شعار). هیچ خط تازه‌ای اضافه نمی‌شود.</p>
  <div class="grid g2" style="margin-top:24px">
    <div class="card">
      <p class="small" style="margin:0 0 6px">استداد — رابط کاربری</p>
      <div style="font-size:40px;font-weight:730;line-height:1.3">با همراه سلام کن.</div>
      <div style="font-size:16px;margin-top:8px">همراه، هوشی مصنوعی برای سلامت روان و حمایت عاطفی است.</div>
      <div class="small" style="margin-top:14px;direction:ltr;text-align:left">Estedad Variable · 390 / 470 / 610 / 730</div>
    </div>
    <div class="card">
      <p class="small" style="margin:0 0 6px">نسخ — نام و لحظه‌های آرام</p>
      <div style="font-family:Naskh;font-size:56px;line-height:1.2">همراه</div>
      <div style="font-family:Naskh;font-size:26px;margin-top:6px">عصر بخیر، محمدمهدی</div>
      <div class="small" style="margin-top:14px;direction:ltr;text-align:left">Noto Naskh Arabic Variable</div>
    </div>
  </div>
</section>`;

/* ---------- 11 voice ---------- */
const voice = `
<section>
  <p class="eyebrow">10 — voice</p>
  <h2>چطور حرف می‌زند</h2>
  <p class="lead">همراه کوتاه می‌نویسد، سؤال نمی‌پیچد و وعده نمی‌دهد. جمله‌ها آرام‌اند، اما نه پنبه‌ای.
  هیچ‌وقت ادعای درمان، تشخیص یا «همیشه کنارتم» نمی‌کند.</p>
  <div class="grid g2" style="margin-top:22px">
    <div class="card do"><h3>می‌گوید</h3><p class="small" style="margin:0">«این را شنیدم. کجایش برایت سخت‌تر بود؟»<br>«می‌توانیم همین‌جا بمانیم یا کمی جلوتر برویم.»<br>«اگر الان حالش را نداری، رد کن؛ بعداً هم می‌شود.»</p></div>
    <div class="card dont"><h3>نمی‌گوید</h3><p class="small" style="margin:0">«تو قطعاً افسرده‌ای.» · «من همیشه حالت را خوب می‌کنم.» · «۶۰۰۰ نفر به من پنج ستاره داده‌اند.»</p></div>
  </div>
</section>`;

/* ---------- 12 motion ---------- */
const motion = `
<section>
  <p class="eyebrow">11 — motion</p>
  <h2>حرکت، به اندازهٔ نفس</h2>
  <p class="lead">سنگ‌ها آرام‌اند. حرکت فقط در دو جا مجاز است: <b>ورود صحنه</b> (بالا آمدن ۴ پیکسلی، ۲۰۰ میلی‌ثانیه)
  و <b>کورسو</b> (تپش آرام ۳ ثانیه‌ای، فقط وقتی همراه در حال پاسخ است). با
  <span class="kbd">prefers-reduced-motion</span> هر دو خاموش می‌شوند.</p>
  <div class="grid g3" style="margin-top:22px">
    <div class="card"><h3>۲۰۰ms</h3><p class="small" style="margin:0">ورود صفحه — همان مقدارِ امروز محصول، دست‌نخورده.</p></div>
    <div class="card"><h3>3s</h3><p class="small" style="margin:0">تپش کورسو هنگام انتظار پاسخ. هیچ حرکت بی‌دلیل دیگری نیست.</p></div>
    <div class="card"><h3>0ms</h3><p class="small" style="margin:0">در حالت کاهش حرکت، همه‌چیز ساکن است — از جمله نشان.</p></div>
  </div>
</section>`;

/* ---------- 13 assets ---------- */
const assets = `
<section>
  <p class="eyebrow">12 — handover</p>
  <h2>فایل‌ها و کد</h2>
  <p class="lead">همهٔ هندسه در <span class="mono" style="display:inline">src/brand/</span> تولید می‌شود؛ تصویرهای ثابت از همان کد بیرون آمده‌اند،
  پس هرگز از هم جدا نمی‌افتند.</p>
  <div class="card" style="margin-top:22px">
  <table>
    <thead><tr><th>فایل</th><th>کاربرد</th><th>وضعیت</th></tr></thead>
    <tbody>
      <tr><td class="mono">src/brand/geometry.js</td><td>تولید سنگ‌ها و خطوط (قطعی و بی‌تصادف)</td><td><span class="pill">منبع حقیقت</span></td></tr>
      <tr><td class="mono">src/brand/scenes.js</td><td>ترکیب صحنه‌ها: خوش‌آمد، حریم، آماده، روز، بینش، نشان</td><td><span class="pill">منبع حقیقت</span></td></tr>
      <tr><td class="mono">src/brand/Mark.jsx</td><td>کامپوننت‌های نشان، کاشی و قفل‌واژه</td><td><span class="pill">React</span></td></tr>
      <tr><td class="mono">public/brand/hamrah-mark.svg</td><td>نشان، جوهر روی مرمر</td><td><span class="pill">ثابت</span></td></tr>
      <tr><td class="mono">public/brand/hamrah-mark-reverse.svg</td><td>نشان، مرمر روی تیره</td><td><span class="pill">ثابت</span></td></tr>
      <tr><td class="mono">public/brand/hamrah-icon.svg</td><td>آیکون اپ (۵۱۲)</td><td><span class="pill">ثابت</span></td></tr>
      <tr><td class="mono">public/favicon.svg</td><td>فاوآیکون</td><td><span class="pill">ثابت</span></td></tr>
    </tbody>
  </table>
  </div>
  <p class="small" style="margin-top:18px">تصویرهای گل‌مانندِ پیشین حذف نشده‌اند؛ در پوشهٔ دارایی‌ها نگه داشته می‌شوند تا مقایسه و بازگشت ممکن بماند.</p>
</section>`;

const book = head + cover + idea + anatomy + sizes + lockup + color + characters + signature + poseSheetSection() + scenes + product + type + voice + motion + assets + foot;

// 1. the copy served by the app: fonts loaded from ./fonts/*.woff2 beside it
fs.writeFileSync(liveBook, book);

// 2. the archived copy: same document with the fonts inlined, so it opens from disk
//    (and inside a sandboxed iframe with no network) without any sibling files.
const inlined = book.replace(/url\(fonts\/([\w-]+\.woff2)\)/g, (_, file) => {
  const bytes = fs.readFileSync(join(fontDir, file));
  return `url(data:font/woff2;base64,${bytes.toString('base64')})`;
});
if (inlined === book) throw new Error('brand book: no @font-face urls were inlined — check the font paths');
fs.writeFileSync(archivedBook, inlined);

console.log('brand book written', fs.statSync(liveBook).size, 'bytes live /', fs.statSync(archivedBook).size, 'bytes self-contained');
