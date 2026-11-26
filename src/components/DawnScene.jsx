import React from 'react';
import SvgTree from './SvgTree';
import { buildDawnScene } from '../art/dawnScene';

/** The six movements of the onboarding story, in route order. */
export const DAWN_STAGES = Object.freeze({
  welcome: 0.06,
  simpleIntro: 0.3,
  deeperIntro: 0.52,
  knowIntro: 0.72,
  almostIntro: 0.88,
  ready: 1,
});

/**
 * سپیدهٔ درون — The Inner Dawn. One continuous scene whose light rises with the
 * user's progress through onboarding. Decorative by default; pass `label` to
 * give it a spoken description.
 */
export default function DawnScene({ progress = DAWN_STAGES.welcome, label, className = '' }) {
  const tree = React.useMemo(() => buildDawnScene(progress), [progress]);
  return <SvgTree tree={tree} className={`dawn-scene ${className}`.trim()} ariaLabel={label} />;
}
