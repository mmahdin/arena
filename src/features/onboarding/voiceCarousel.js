// The voice deck is physically left-to-right: a leftward drag reveals the
// next voice, regardless of the RTL direction of its Persian card copy.
export function resistVoiceDrag(dx, index, count) {
  if ((index === 0 && dx > 0) || (index === count - 1 && dx < 0)) {
    return Math.sign(dx) * Math.min(Math.abs(dx) * 0.22, 48);
  }
  return dx;
}

export function voiceSwipeTarget({ index, count, dx, velocity, step }) {
  const span = Math.max(1, step);
  const farEnough = Math.abs(dx) > span * 0.20;
  const quickFlick = Math.abs(dx) > 18 && Math.abs(velocity) > 0.4;
  if (!farEnough && !quickFlick) return index;

  // Velocity contributes a brief, bounded coast. Long drags can skip a card,
  // but a single flick never jumps unpredictably across the entire deck.
  const coast = Math.max(-span * 0.7, Math.min(span * 0.7, velocity * 130));
  const projected = dx + coast;
  const direction = dx < 0 ? 1 : -1;
  const steps = Math.min(2, Math.max(1, Math.round(Math.abs(projected) / span)));
  return Math.max(0, Math.min(count - 1, index + direction * steps));
}
