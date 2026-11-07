/** The five conversation approaches map one-to-one to persistent backend values. */
export const APPROACH_TO_STYLE = Object.freeze({
  'گفت‌وگوی آزاد': 'balanced',
  'همدلی و شنیدن': 'listening',
  'به من یاد بده': 'teaching',
  'راهنمایی آرام': 'gentle_guidance',
  'حل مسئله با هم': 'collaborative',
});

export const STYLE_TO_APPROACH = Object.freeze({
  balanced: 'گفت‌وگوی آزاد',
  listening: 'همدلی و شنیدن',
  teaching: 'به من یاد بده',
  gentle_guidance: 'راهنمایی آرام',
  collaborative: 'حل مسئله با هم',
  guidance: 'راهنمایی آرام', // previous three-style profiles
});

export const PROFILE_STYLES = Object.freeze(Object.keys(STYLE_TO_APPROACH));

export function styleForApproach(approach) {
  return APPROACH_TO_STYLE[approach] || 'balanced';
}

export function approachForStyle(style) {
  return STYLE_TO_APPROACH[style] || STYLE_TO_APPROACH.balanced;
}

export function validateDisplayName(input) {
  const name = typeof input === 'string' ? input.trim() : '';
  return [...name].length >= 1 && [...name].length <= 60 ? name : null;
}
