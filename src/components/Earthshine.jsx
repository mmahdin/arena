import React, { useId, useRef } from 'react';
import { clampAperture, crescentPath, EARTHSHINE_SCENES } from '../features/onboarding/earthshineModel';

export { clampAperture, crescentPath, EARTHSHINE_SCENES };

/**
 * Renders the Moon as a physical, earthlit sphere:
 * - Always-present desaturated blue-gray ashen disc (gradually revealed by `earthshine`)
 * - Subtle blurred lunar maria (seas) clipped strictly inside the lunar limb
 * - Soft penumbral transition along the terminator (no hard vector outline)
 * - Cool-white sunlit waning crescent with a restrained atmospheric glare halo
 */
export function EarthshineMoon({
  idPrefix,
  cx,
  cy,
  r,
  phase = 0.21,
  tiltDeg = 23,
  earthshine = 0.5,
  maria = 0.5,
  halo = 0.06,
  reversed = false,
  skyBase = '#0B1528',
}) {
  const clipId = `${idPrefix}-limb`;
  const ashenGradId = `${idPrefix}-ashen`;
  const mariaBlurId = `${idPrefix}-maria`;
  const penumbraBlurId = `${idPrefix}-penumbra`;
  const haloGradId = `${idPrefix}-halo`;

  const litColor = reversed ? '#F6F8FC' : '#EEF3FA';
  const ashenLight = reversed ? '#879BB3' : '#7389A6';
  const ashenMid = reversed ? '#93A6BC' : '#586D89';
  const ashenDeep = reversed ? '#9FB0C4' : '#3E516B';
  const mariaTone = reversed ? '#6B7F98' : '#26364D';

  const rad = (tiltDeg * Math.PI) / 180;
  const limbX = cx + r * 0.72 * Math.cos(rad);
  const limbY = cy + r * 0.72 * Math.sin(rad);

  const discOpacity = reversed
    ? 0.56
    : Math.max(0.055, Math.min(0.92, earthshine * 0.84));
  const mariaOpacity = Math.max(0, Math.min(1, maria * earthshine)) * (reversed ? 0.24 : 0.34);

  const mainCrescent = crescentPath(cx, cy, r, phase, tiltDeg);
  const penumbraSteps = [
    { dp: 0.018, op: 0.34 },
    { dp: 0.042, op: 0.18 },
    { dp: 0.072, op: 0.09 },
    { dp: 0.108, op: 0.04 },
  ];

  const mariaPath = [
    `M ${(cx - r * 0.62).toFixed(2)} ${(cy - r * 0.16).toFixed(2)}`,
    `C ${(cx - r * 0.68).toFixed(2)} ${(cy - r * 0.48).toFixed(2)}, ${(cx - r * 0.28).toFixed(2)} ${(cy - r * 0.64).toFixed(2)}, ${(cx + r * 0.04).toFixed(2)} ${(cy - r * 0.46).toFixed(2)}`,
    `C ${(cx + r * 0.34).toFixed(2)} ${(cy - r * 0.38).toFixed(2)}, ${(cx + r * 0.46).toFixed(2)} ${(cy - r * 0.06).toFixed(2)}, ${(cx + r * 0.24).toFixed(2)} ${(cy + r * 0.16).toFixed(2)}`,
    `C ${(cx + r * 0.06).toFixed(2)} ${(cy + r * 0.32).toFixed(2)}, ${(cx - r * 0.24).toFixed(2)} ${(cy + r * 0.44).toFixed(2)}, ${(cx - r * 0.48).toFixed(2)} ${(cy + r * 0.26).toFixed(2)}`,
    `C ${(cx - r * 0.66).toFixed(2)} ${(cy + r * 0.12).toFixed(2)}, ${(cx - r * 0.56).toFixed(2)} ${(cy - r * 0.02).toFixed(2)}, ${(cx - r * 0.62).toFixed(2)} ${(cy - r * 0.16).toFixed(2)} Z`,
  ].join(' ');
  const crisiumCx = cx + r * 0.44;
  const crisiumCy = cy - r * 0.26;

  return (
    <g className="es-moon">
      <defs>
        <clipPath id={clipId}>
          <circle cx={cx} cy={cy} r={r} />
        </clipPath>

        <radialGradient
          id={ashenGradId}
          cx="62%"
          cy="56%"
          r="72%"
          fx="74%"
          fy="62%"
        >
          <stop offset="0%" stopColor={ashenLight} />
          <stop offset="56%" stopColor={ashenMid} />
          <stop offset="100%" stopColor={ashenDeep} />
        </radialGradient>

        <filter id={mariaBlurId} x="-35%" y="-35%" width="170%" height="170%">
          <feGaussianBlur stdDeviation={Math.max(1.3, r * 0.044)} />
        </filter>

        <filter id={penumbraBlurId} x="-35%" y="-35%" width="170%" height="170%">
          <feGaussianBlur stdDeviation={Math.max(0.95, r * 0.036)} />
        </filter>

        <radialGradient
          id={haloGradId}
          gradientUnits="userSpaceOnUse"
          cx={limbX}
          cy={limbY}
          r={r * 2.35}
        >
          <stop offset="0%" stopColor={litColor} stopOpacity={halo} />
          <stop offset="36%" stopColor={litColor} stopOpacity={halo * 0.36} />
          <stop offset="72%" stopColor={litColor} stopOpacity={halo * 0.08} />
          <stop offset="100%" stopColor={litColor} stopOpacity="0" />
        </radialGradient>
      </defs>

      {halo > 0.005 && (
        <circle
          cx={limbX}
          cy={limbY}
          r={r * 2.35}
          fill={`url(#${haloGradId})`}
          pointerEvents="none"
        />
      )}

      {!reversed && <circle cx={cx} cy={cy} r={r} fill={skyBase} />}

      <circle
        cx={cx}
        cy={cy}
        r={r}
        fill={`url(#${ashenGradId})`}
        opacity={discOpacity}
      />

      {mariaOpacity > 0.004 && (
        <g clipPath={`url(#${clipId})`} opacity={mariaOpacity} filter={`url(#${mariaBlurId})`}>
          <path d={mariaPath} fill={mariaTone} />
          <ellipse
            cx={crisiumCx}
            cy={crisiumCy}
            rx={r * 0.11}
            ry={r * 0.14}
            transform={`rotate(24 ${crisiumCx.toFixed(2)} ${crisiumCy.toFixed(2)})`}
            fill={mariaTone}
            opacity="0.72"
          />
          <ellipse
            cx={cx - r * 0.14}
            cy={cy - r * 0.24}
            rx={r * 0.19}
            ry={r * 0.15}
            fill={mariaTone}
            opacity="0.48"
          />
        </g>
      )}

      <g clipPath={`url(#${clipId})`} filter={`url(#${penumbraBlurId})`}>
        {penumbraSteps.map((step) => (
          <path
            key={step.dp}
            d={crescentPath(cx, cy, r * 1.005, Math.min(0.46, phase + step.dp), tiltDeg)}
            fill={litColor}
            opacity={reversed ? step.op * 0.65 : step.op}
          />
        ))}
      </g>

      <g clipPath={`url(#${clipId})`}>
        <path
          d={mainCrescent}
          fill={litColor}
          opacity={reversed ? 0.92 : 0.98}
        />
      </g>
    </g>
  );
}

/**
 * Full-bleed vertical sky scene for the eight Earthshine onboarding moments.
 */
export function EarthshineScene({
  scene = 'welcome',
  aperture = 0.22,
  onApertureChange,
  venusVisible = true,
  className = '',
}) {
  const uid = useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const stageRef = useRef(null);
  const draggingRef = useRef(false);

  const config = EARTHSHINE_SCENES[scene] || EARTHSHINE_SCENES.welcome;
  const W = 390;
  const H = 480;
  const cx = W * config.moonCx;
  const cy = H * config.moonCy;
  const r = W * config.moonR;
  const horizonY = config.horizon ? H * config.horizonY : H;

  const skyGradId = `es-sky-${uid}`;
  const seaGradId = `es-sea-${uid}`;
  const falseDawnId = `es-fdawn-${uid}`;
  const solarRimId = `es-srim-${uid}`;
  const solarCoreId = `es-score-${uid}`;
  const venusGlowId = `es-vglow-${uid}`;
  const slitClipId = `es-slit-${uid}`;
  const grainFilterId = `es-grain-${uid}`;

  const clampedAperture = clampAperture(aperture);
  const minSlitW = 24;
  const maxSlitW = W * 0.78;
  const slitW = minSlitW + (maxSlitW - minSlitW) * ((clampedAperture - 0.08) / 0.84);
  const slitCenter = config.shutter
    ? cx + r * 0.68 * (1 - (clampedAperture - 0.08) / 0.84)
    : W / 2;
  const slitX = Math.max(12, Math.min(W - 12 - slitW, slitCenter - slitW / 2));

  const updateApertureFromPointer = (clientX) => {
    if (!onApertureChange || !stageRef.current) return;
    const rect = stageRef.current.getBoundingClientRect();
    if (!rect.width) return;
    const distFromCenter = Math.abs(clientX - (rect.left + rect.width / 2));
    const normalized = Math.min(1, (distFromCenter / (rect.width * 0.42)));
    onApertureChange(clampAperture(0.10 + normalized * 0.82));
  };

  const interactiveShutter = Boolean(config.shutter && onApertureChange);

  const skyBody = (
    <>
      <rect x="0" y="0" width={W} height={horizonY + 1} fill={`url(#${skyGradId})`} />

      {config.falseDawn && (
        <ellipse
          cx={W * 0.54}
          cy={horizonY}
          rx={W * 0.48}
          ry={H * 0.44}
          fill={`url(#${falseDawnId})`}
          className="es-false-dawn"
        />
      )}

      {config.solarRim && (
        <g className="es-solar-rim">
          <ellipse
            cx={W * config.solarRim.x}
            cy={horizonY}
            rx={W * 0.46}
            ry={H * 0.24}
            fill={`url(#${solarRimId})`}
          />
          <ellipse
            cx={W * config.solarRim.x}
            cy={horizonY + 0.4}
            rx={22}
            ry={4.5}
            fill={`url(#${solarCoreId})`}
          />
        </g>
      )}

      {config.stars.length > 0 && (
        <g className="es-stars">
          {config.stars.map((star, idx) => (
            <circle
              key={idx}
              cx={W * star.x}
              cy={H * star.y}
              r={star.r}
              fill="#DCE6F5"
              style={{
                '--star-opacity': star.opacity,
                '--star-delay': star.delay,
              }}
              opacity={star.opacity}
            />
          ))}
        </g>
      )}

      {config.venus > 0 && (
        <g
          className={`es-venus ${venusVisible ? 'is-lit' : 'is-faded'}`}
          style={{ opacity: venusVisible ? config.venus : 0 }}
        >
          <circle
            cx={W * config.venusPos.x}
            cy={H * config.venusPos.y}
            r={config.reversed ? 5.5 : 8}
            fill={`url(#${venusGlowId})`}
          />
          <circle
            cx={W * config.venusPos.x}
            cy={H * config.venusPos.y}
            r={config.reversed ? 1.15 : 1.45}
            fill={config.reversed ? '#FAF5EC' : '#FFF7EA'}
          />
        </g>
      )}

      <EarthshineMoon
        idPrefix={`es-moon-${uid}`}
        cx={cx}
        cy={cy}
        r={r}
        phase={config.phase}
        tiltDeg={config.tiltDeg}
        earthshine={config.earthshine}
        maria={config.maria}
        halo={config.halo}
        reversed={config.reversed}
        skyBase={config.skyStops[1]?.[1] || '#0B1528'}
      />

      {config.horizon && (
        <g className="es-horizon">
          <rect
            x="0"
            y={horizonY}
            width={W}
            height={H - horizonY + 2}
            fill={`url(#${seaGradId})`}
          />
          <line
            x1="0"
            y1={horizonY}
            x2={W}
            y2={horizonY}
            stroke={config.horizonLine}
            strokeWidth="0.9"
          />
        </g>
      )}
    </>
  );

  return (
    <div
      ref={stageRef}
      className={`es-stage es-stage--${config.key} ${interactiveShutter ? 'es-stage--interactive' : ''} ${className}`.trim()}
      onPointerDown={interactiveShutter ? (event) => {
        if (!event.isPrimary) return;
        draggingRef.current = true;
        event.currentTarget.setPointerCapture?.(event.pointerId);
        updateApertureFromPointer(event.clientX);
      } : undefined}
      onPointerMove={interactiveShutter ? (event) => {
        if (!draggingRef.current) return;
        updateApertureFromPointer(event.clientX);
      } : undefined}
      onPointerUp={interactiveShutter ? (event) => {
        draggingRef.current = false;
        if (event.currentTarget.hasPointerCapture?.(event.pointerId)) {
          event.currentTarget.releasePointerCapture(event.pointerId);
        }
      } : undefined}
      onPointerCancel={interactiveShutter ? () => { draggingRef.current = false; } : undefined}
    >
      <svg
        className="es-canvas"
        viewBox={`0 0 ${W} ${H}`}
        preserveAspectRatio="xMidYMid slice"
        role="img"
        aria-label={config.label}
      >
        <defs>
          <linearGradient id={skyGradId} x1="0" y1="0" x2="0" y2="1">
            {config.skyStops.map(([offset, color]) => (
              <stop key={offset} offset={`${offset * 100}%`} stopColor={color} />
            ))}
          </linearGradient>

          {config.horizon && (
            <linearGradient id={seaGradId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={config.seaTop} />
              <stop offset="100%" stopColor={config.seaBottom} />
            </linearGradient>
          )}

          {config.falseDawn && (
            <radialGradient id={falseDawnId} cx="50%" cy="100%" r="95%">
              <stop offset="0%" stopColor="#CFA79B" stopOpacity="0.24" />
              <stop offset="46%" stopColor="#9D8387" stopOpacity="0.11" />
              <stop offset="100%" stopColor="#2A344B" stopOpacity="0" />
            </radialGradient>
          )}

          {config.solarRim && (
            <>
              <radialGradient id={solarRimId} cx="50%" cy="100%" r="95%">
                <stop offset="0%" stopColor="#F5D39E" stopOpacity="0.52" />
                <stop offset="42%" stopColor="#DDA874" stopOpacity="0.20" />
                <stop offset="100%" stopColor="#586D88" stopOpacity="0" />
              </radialGradient>
              <radialGradient id={solarCoreId} cx="50%" cy="65%" r="65%">
                <stop offset="0%" stopColor="#FFF6DF" stopOpacity="0.95" />
                <stop offset="55%" stopColor="#F2C88B" stopOpacity="0.55" />
                <stop offset="100%" stopColor="#F2C88B" stopOpacity="0" />
              </radialGradient>
            </>
          )}

          {config.venus > 0 && (
            <radialGradient id={venusGlowId} cx="50%" cy="50%" r="50%">
              <stop offset="0%" stopColor="#FFF4E0" stopOpacity={config.reversed ? '0.26' : '0.38'} />
              <stop offset="48%" stopColor="#F5DFBF" stopOpacity={config.reversed ? '0.08' : '0.12'} />
              <stop offset="100%" stopColor="#F5DFBF" stopOpacity="0" />
            </radialGradient>
          )}

          {config.shutter && (
            <clipPath id={slitClipId}>
              <rect x={slitX} y="0" width={slitW} height={H} />
            </clipPath>
          )}

          <filter id={grainFilterId} x="0%" y="0%" width="100%" height="100%">
            <feTurbulence type="fractalNoise" baseFrequency="0.78" numOctaves="3" seed="11" stitchTiles="stitch" result="noise" />
            <feColorMatrix type="saturate" values="0" in="noise" result="mono" />
            <feComponentTransfer in="mono">
              <feFuncA type="linear" slope={config.reversed ? '0.024' : '0.032'} />
            </feComponentTransfer>
          </filter>
        </defs>

        {config.shutter ? (
          <>
            <rect x="0" y="0" width={W} height={H} fill={config.shutterColor} />
            <g clipPath={`url(#${slitClipId})`}>{skyBody}</g>
            <line
              x1={slitX}
              y1="0"
              x2={slitX}
              y2={H}
              stroke="rgba(182, 198, 222, 0.15)"
              strokeWidth="0.75"
            />
            <line
              x1={slitX + slitW}
              y1="0"
              x2={slitX + slitW}
              y2={H}
              stroke="rgba(182, 198, 222, 0.15)"
              strokeWidth="0.75"
            />
          </>
        ) : skyBody}

        <rect
          x="0"
          y="0"
          width={W}
          height={H}
          fill="#FFFFFF"
          filter={`url(#${grainFilterId})`}
          pointerEvents="none"
        />
      </svg>
    </div>
  );
}

export default EarthshineScene;
