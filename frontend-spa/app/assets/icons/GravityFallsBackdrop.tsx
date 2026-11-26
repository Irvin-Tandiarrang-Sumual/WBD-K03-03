import type { ScenicSceneId } from '~/types/ui';

export interface GravityFallsBackdropProps {
  scene: ScenicSceneId;
  dense?: boolean;
}

const renderOrganicPinePath = (
  x: number,
  foliageBaseY: number,
  foliageHeight: number,
  width: number
): string => {
  const top = foliageBaseY - foliageHeight;
  const tier1Y = top + foliageHeight * 0.28;
  const tier2Y = top + foliageHeight * 0.56;
  const tier3Y = top + foliageHeight * 0.82;
  const w1 = width * 0.28;
  const w2 = width * 0.42;
  const w3 = width * 0.54;
  const wBase = width * 0.58;

  return [
    `M ${x} ${top}`,
    `Q ${x - w1 * 0.45} ${top + foliageHeight * 0.16} ${x - w1} ${tier1Y}`,
    `Q ${x - w1 * 0.55} ${tier1Y + foliageHeight * 0.03} ${x - w1 * 0.48} ${tier1Y + foliageHeight * 0.01}`,
    `Q ${x - w2 * 0.72} ${top + foliageHeight * 0.44} ${x - w2} ${tier2Y}`,
    `Q ${x - w2 * 0.62} ${tier2Y + foliageHeight * 0.04} ${x - w2 * 0.54} ${tier2Y + foliageHeight * 0.01}`,
    `Q ${x - w3 * 0.76} ${top + foliageHeight * 0.72} ${x - w3} ${tier3Y}`,
    `Q ${x - w3 * 0.7} ${tier3Y + foliageHeight * 0.03} ${x - w3 * 0.62} ${tier3Y + foliageHeight * 0.01}`,
    `Q ${x - wBase * 0.85} ${foliageBaseY - foliageHeight * 0.06} ${x - wBase} ${foliageBaseY}`,
    `L ${x + wBase} ${foliageBaseY}`,
    `Q ${x + wBase * 0.85} ${foliageBaseY - foliageHeight * 0.06} ${x + w3 * 0.62} ${tier3Y + foliageHeight * 0.01}`,
    `Q ${x + w3 * 0.7} ${tier3Y + foliageHeight * 0.03} ${x + w3} ${tier3Y}`,
    `Q ${x + w3 * 0.76} ${top + foliageHeight * 0.72} ${x + w2 * 0.54} ${tier2Y + foliageHeight * 0.01}`,
    `Q ${x + w2 * 0.62} ${tier2Y + foliageHeight * 0.04} ${x + w2} ${tier2Y}`,
    `Q ${x + w2 * 0.72} ${top + foliageHeight * 0.44} ${x + w1 * 0.48} ${tier1Y + foliageHeight * 0.01}`,
    `Q ${x + w1 * 0.55} ${tier1Y + foliageHeight * 0.03} ${x + w1} ${tier1Y}`,
    `Q ${x + w1 * 0.45} ${top + foliageHeight * 0.16} ${x} ${top}`,
    'Z',
  ].join(' ');
};

interface OrganicPineTreeProps {
  x: number;
  groundY: number;
  height: number;
  width: number;
  trunkHeight?: number;
  foliageFill?: string;
  trunkFill?: string;
}

const OrganicPineTree = ({
  x,
  groundY,
  height,
  width,
  trunkHeight = 28,
  foliageFill = '#264D3B',
  trunkFill = '#3A241A',
}: OrganicPineTreeProps) => {
  const safeTrunkH = Math.max(18, trunkHeight);
  const foliageBaseY = groundY - safeTrunkH;
  const foliageHeight = Math.max(48, height - safeTrunkH);
  const trunkWidth = Math.max(6, Math.round(width * 0.14));

  return (
    <g data-pine-tree="true">
      <rect
        x={x - trunkWidth / 2}
        y={foliageBaseY - 6}
        width={trunkWidth}
        height={safeTrunkH + 18}
        rx={1.5}
        fill={trunkFill}
      />
      <path d={renderOrganicPinePath(x, foliageBaseY, foliageHeight, width)} fill={foliageFill} />
    </g>
  );
};

export const GravityFallsBackdrop = ({
  scene,
  dense = false,
}: GravityFallsBackdropProps) => {
  const silhouetteOpacity = dense ? 0.2 : 0.16;
  const midOpacity = dense ? 0.13 : 0.11;
  const farOpacity = dense ? 0.1 : 0.08;

  return (
    <div
      data-scene={scene}
      className="fixed inset-0 pointer-events-none select-none overflow-hidden z-0"
      aria-hidden="true"
    >
      <div className="absolute -top-28 right-1/4 w-[38rem] h-[22rem] rounded-full bg-[#F3B738]/12 blur-[115px]" />
      <div className="absolute top-1/3 -left-24 w-[28rem] h-[26rem] rounded-full bg-[#1E6B8C]/8 blur-[125px]" />

      <div className="absolute inset-0 opacity-15 journal-dot-grid" />

      {scene === 'valley_trestle' && (
        <svg
          className={`absolute inset-x-0 bottom-0 w-full ${
            dense ? 'h-[48vh] min-h-[290px]' : 'h-[46vh] min-h-[270px]'
          }`}
          viewBox="0 0 1440 400"
          preserveAspectRatio="xMidYMax slice"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            <linearGradient id="valley-mountain-mist" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#264D3B" stopOpacity={farOpacity} />
              <stop offset="100%" stopColor="#264D3B" stopOpacity="0.02" />
            </linearGradient>
            <linearGradient id="trestle-mist-stroke" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#264D3B" stopOpacity={silhouetteOpacity + 0.04} />
              <stop offset="68%" stopColor="#264D3B" stopOpacity={midOpacity} />
              <stop offset="100%" stopColor="#264D3B" stopOpacity="0.02" />
            </linearGradient>
            <linearGradient id="valley-floor-haze" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#F5EEDD" stopOpacity="0" />
              <stop offset="65%" stopColor="#F5EEDD" stopOpacity="0.52" />
              <stop offset="100%" stopColor="#F5EEDD" stopOpacity="0.18" />
            </linearGradient>
          </defs>

          <path
            d="M0 225 C155 138, 285 118, 435 174 C575 224, 695 96, 865 136 C1025 174, 1165 104, 1315 152 C1375 172, 1410 162, 1440 168 L1440 400 L0 400 Z"
            fill="url(#valley-mountain-mist)"
          />

          <g stroke="url(#trestle-mist-stroke)" fill="none" strokeLinecap="round">
            <path d="M180 250 Q720 236 1260 250" strokeWidth="5.5" />
            <path d="M190 259 Q720 245 1250 259" strokeWidth="2.5" />
            <path d="M295 304 Q720 294 1145 304" strokeWidth="1.8" />
            {[290, 395, 500, 605, 710, 815, 920, 1025, 1130].map((x, idx) => (
              <g key={x}>
                <path d={`M ${x} 248 Q ${x - 4} 315 ${x - 10} 388`} strokeWidth="3" />
                <path d={`M ${x + 14} 248 Q ${x + 18} 315 ${x + 24} 388`} strokeWidth="3" />
                <line x1={x} y1="258" x2={x + 18} y2="302" strokeWidth="1.3" />
                <line x1={x + 14} y1="258" x2={x - 4} y2="302" strokeWidth="1.3" />
                {idx < 8 && (
                  <path d={`M ${x + 16} 276 Q ${x + 60} 252 ${x + 105} 276`} strokeWidth="1.6" />
                )}
              </g>
            ))}
          </g>

          <path
            d="M0 305 C320 278, 620 326, 940 302 C1180 284, 1340 298, 1440 292 L1440 400 L0 400 Z"
            fill="url(#valley-floor-haze)"
          />

          <g opacity={silhouetteOpacity + 0.04}>
            {[
              { x: 42, groundY: 318, h: 176, w: 56, trunk: 34 },
              { x: 112, groundY: 326, h: 198, w: 64, trunk: 38 },
              { x: 186, groundY: 344, h: 162, w: 50, trunk: 34 },
              { x: 1252, groundY: 346, h: 164, w: 52, trunk: 34 },
              { x: 1326, groundY: 324, h: 202, w: 64, trunk: 38 },
              { x: 1398, groundY: 316, h: 184, w: 58, trunk: 34 },
            ].map((t, i) => (
              <OrganicPineTree
                key={`valley-pine-${i}`}
                x={t.x}
                groundY={t.groundY}
                height={t.h}
                width={t.w}
                trunkHeight={t.trunk}
              />
            ))}
          </g>

          <path
            d="M0 286 C125 288, 245 310, 365 352 C435 376, 490 388, 540 400 L0 400 Z"
            fill="#264D3B"
            fillOpacity={silhouetteOpacity + 0.04}
          />
          <path
            d="M1440 284 C1310 286, 1185 310, 1065 352 C995 376, 940 388, 895 400 L1440 400 Z"
            fill="#264D3B"
            fillOpacity={silhouetteOpacity + 0.04}
          />

          <path
            d="M460 382 C580 356, 680 374, 770 364 C860 354, 935 370, 990 382 L990 400 L460 400 Z"
            fill="#1E6B8C"
            fillOpacity="0.1"
          />
        </svg>
      )}

      {scene === 'pine_forest' && (
        <svg
          className={`absolute inset-x-0 bottom-0 w-full ${
            dense ? 'h-[48vh] min-h-[290px]' : 'h-[45vh] min-h-[265px]'
          }`}
          viewBox="0 0 1440 400"
          preserveAspectRatio="xMidYMax slice"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            <linearGradient id="gf-pine-mountain-fade" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#264D3B" stopOpacity={farOpacity} />
              <stop offset="72%" stopColor="#264D3B" stopOpacity={farOpacity * 0.55} />
              <stop offset="100%" stopColor="#264D3B" stopOpacity="0.02" />
            </linearGradient>
            <linearGradient id="gf-valley-mist-band" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#F5EEDD" stopOpacity="0" />
              <stop offset="52%" stopColor="#F5EEDD" stopOpacity="0.45" />
              <stop offset="100%" stopColor="#F5EEDD" stopOpacity="0" />
            </linearGradient>
          </defs>

          <path
            d="M0 245 C140 165, 255 138, 395 184 C525 228, 635 108, 795 142 C945 174, 1070 96, 1225 148 C1325 182, 1385 152, 1440 168 L1440 400 L0 400 Z"
            fill="url(#gf-pine-mountain-fade)"
          />

          <path
            d="M0 284 C190 234, 340 248, 520 280 C710 314, 870 238, 1080 258 C1235 272, 1345 234, 1440 250 L1440 400 L0 400 Z"
            fill="#264D3B"
            fillOpacity={midOpacity}
          />

          <path
            d="M0 266 C280 238, 540 286, 820 260 C1080 236, 1280 264, 1440 246 L1440 340 L0 340 Z"
            fill="url(#gf-valley-mist-band)"
          />

          <g opacity={midOpacity + 0.03}>
            {[
              { x: 64, h: 162, w: 52, trunk: 24 },
              { x: 146, h: 138, w: 46, trunk: 22 },
              { x: 248, h: 172, w: 56, trunk: 26 },
              { x: 356, h: 130, w: 42, trunk: 20 },
              { x: 478, h: 114, w: 38, trunk: 20 },
              { x: 610, h: 102, w: 34, trunk: 18 },
              { x: 752, h: 98, w: 34, trunk: 18 },
              { x: 896, h: 118, w: 40, trunk: 20 },
              { x: 1032, h: 144, w: 46, trunk: 22 },
              { x: 1152, h: 168, w: 54, trunk: 26 },
              { x: 1272, h: 152, w: 48, trunk: 24 },
              { x: 1378, h: 178, w: 58, trunk: 28 },
            ].map((t, i) => (
              <OrganicPineTree
                key={`forest-mid-${i}`}
                x={t.x}
                groundY={400}
                height={t.h}
                width={t.w}
                trunkHeight={t.trunk}
              />
            ))}
          </g>

          <path
            d="M0 352 C260 324, 520 364, 780 352 C1040 340, 1260 318, 1440 334 L1440 400 L0 400 Z"
            fill="#264D3B"
            fillOpacity={silhouetteOpacity * 0.8}
          />

          <g opacity={silhouetteOpacity + 0.04}>
            {[
              { x: 28, h: 222, w: 68, trunk: 34 },
              { x: 106, h: 184, w: 56, trunk: 28 },
              { x: 196, h: 206, w: 62, trunk: 32 },
              { x: 296, h: 152, w: 46, trunk: 24 },
              { x: 414, h: 120, w: 38, trunk: 20 },
              { x: 968, h: 126, w: 40, trunk: 22 },
              { x: 1094, h: 162, w: 50, trunk: 26 },
              { x: 1208, h: 202, w: 62, trunk: 32 },
              { x: 1314, h: 180, w: 56, trunk: 28 },
              { x: 1408, h: 228, w: 70, trunk: 36 },
            ].map((t, i) => (
              <OrganicPineTree
                key={`forest-fg-${i}`}
                x={t.x}
                groundY={400}
                height={t.h}
                width={t.w}
                trunkHeight={t.trunk}
              />
            ))}
          </g>
        </svg>
      )}
    </div>
  );
};
