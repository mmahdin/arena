const MOOD_ACKNOWLEDGEMENTS = [
  'می‌فهمم که این روزها برات سخت می‌گذره. لازم نیست عجله کنیم.',
  'انگار این روزها انرژی زیادی نداری. می‌تونیم از چیزهای کوچیک شروع کنیم.',
  'انگار بعضی روزها بهتر و بعضی روزها سخت‌تر می‌گذرن.',
  'خوبه که بهم گفتی این روزها چطور می‌گذره.',
  'خوشحالم که در کل حالت خوبه.',
];

/** A client-only, transient welcome; it never enters the persisted message history. */
export function createOnboardingGreeting(name, answers = {}) {
  const displayName = typeof name === 'string' ? name.trim().slice(0, 60) : '';
  const greeting = `سلام${displayName ? ` ${displayName}` : ''}، خوش اومدی.`;
  const mood = Number.isInteger(answers.mood) && answers.mood >= 0 && answers.mood <= 4
    ? MOOD_ACKNOWLEDGEMENTS[answers.mood] : null;
  const concern = Array.isArray(answers.concerns) && typeof answers.concerns[0] === 'string'
    ? answers.concerns[0] : null;
  const focus = Array.isArray(answers.focus) && typeof answers.focus[0] === 'string'
    ? answers.focus[0] : null;
  const lines = [greeting, mood].filter(Boolean);
  if (concern) lines.push(`گفتی «${concern}» این روزها ذهنت رو درگیر کرده.`);
  if (focus) lines.push(`دوست داری روی «${focus}» هم کار کنیم.`);
  lines.push('از کدومش دوست داری شروع کنیم؟ هر چیزی که خودت راحت‌تری.');
  return lines.join('\n\n');
}
