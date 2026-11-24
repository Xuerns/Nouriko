import type { SVGProps } from "react";

export function LoginMealIllustration(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      viewBox="0 0 460 300"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className="w-full max-w-[380px] h-auto select-none"
      {...props}
    >
      {/* Background Soft Organic Aura */}
      <ellipse cx="230" cy="225" rx="190" ry="45" fill="var(--brand-tint)" opacity="0.6" />
      <ellipse cx="215" cy="220" rx="140" ry="32" fill="var(--bg-subtle)" />

      {/* Decorative Floating Leaves */}
      <path
        d="M60 170 C45 150 45 125 70 120 C85 138 85 160 60 170 Z"
        fill="var(--brand-primary)"
        opacity="0.35"
      />
      <path
        d="M65 145 C60 135 70 125 70 120"
        stroke="var(--brand-primary)"
        strokeWidth="1.5"
        strokeLinecap="round"
        opacity="0.5"
      />
      <path
        d="M95 195 C82 180 84 165 102 160 C114 172 112 187 95 195 Z"
        fill="var(--brand-primary)"
        opacity="0.4"
      />
      <path
        d="M395 130 C410 115 410 95 390 90 C375 105 375 122 395 130 Z"
        fill="var(--brand-primary)"
        opacity="0.3"
      />

      {/* Glass of Water */}
      <g transform="translate(305, 135)">
        {/* Water Glass Shadow */}
        <ellipse cx="24" cy="94" rx="22" ry="7" fill="rgba(0,0,0,0.06)" />
        {/* Glass Body */}
        <path
          d="M6 10 L10 88 C10 94 38 94 38 88 L42 10 Z"
          fill="#D6EEF8"
          stroke="#93C5FD"
          strokeWidth="2"
          className="dark:fill-[#1e3a47] dark:stroke-[#38bdf8]"
          opacity="0.85"
        />
        {/* Water Surface Line */}
        <ellipse cx="24" cy="25" rx="16" ry="4" fill="#BAE6FD" opacity="0.9" />
        {/* Glass Rim */}
        <ellipse cx="24" cy="10" rx="18" ry="4.5" fill="none" stroke="#93C5FD" strokeWidth="2" />
        {/* Glass Highlight */}
        <path d="M14 20 L16 80" stroke="#FFFFFF" strokeWidth="2" strokeLinecap="round" opacity="0.7" />
      </g>

      {/* The Food Bowl */}
      <g transform="translate(70, 70)">
        {/* Bowl Shadow */}
        <ellipse cx="140" cy="175" rx="115" ry="24" fill="rgba(0,0,0,0.08)" />

        {/* Bowl Exterior */}
        <path
          d="M25 80 C25 155 80 178 140 178 C200 178 255 155 255 80 Z"
          fill="#F5EFE6"
          stroke="#D8CEBD"
          strokeWidth="2.5"
          className="dark:fill-[#222924] dark:stroke-[#343e37]"
        />

        {/* Bowl Rim */}
        <ellipse
          cx="140"
          cy="80"
          rx="115"
          ry="32"
          fill="#EDE4D5"
          stroke="#D8CEBD"
          strokeWidth="2.5"
          className="dark:fill-[#1b221d] dark:stroke-[#343e37]"
        />

        {/* Food Base: Broth / Grains */}
        <ellipse
          cx="140"
          cy="83"
          rx="105"
          ry="27"
          fill="#F4DEB9"
          className="dark:fill-[#42311f]"
        />

        {/* Savory Sauce Swirl */}
        <path
          d="M60 85 C80 98 120 95 140 90 C180 80 215 90 225 85"
          stroke="#C87A38"
          strokeWidth="10"
          strokeLinecap="round"
          opacity="0.85"
        />

        {/* Fresh Green Herbs & Broccoli / Spinach */}
        <path
          d="M65 72 C55 60 70 50 85 58 C95 48 115 55 110 70 C100 85 75 85 65 72 Z"
          fill="var(--brand-primary)"
        />
        <circle cx="85" cy="62" r="5" fill="#4ADE80" opacity="0.6" />
        <circle cx="100" cy="72" r="4" fill="#86EFAC" opacity="0.6" />

        <path
          d="M175 75 C168 62 185 52 198 60 C210 52 225 60 220 74 C215 88 190 90 175 75 Z"
          fill="var(--brand-primary)"
        />

        {/* Cherry Tomato Slices */}
        <circle cx="178" cy="95" r="14" fill="var(--brand-accent)" />
        <circle cx="178" cy="95" r="10" fill="#E25822" />
        <ellipse cx="178" cy="95" rx="5" ry="3" fill="#F87171" opacity="0.7" />

        {/* Sunny-side-up Egg */}
        <path
          d="M110 82 C100 70 115 55 135 58 C155 52 170 65 165 80 C162 92 145 98 130 96 C115 95 112 88 110 82 Z"
          fill="#FFFFFF"
          stroke="#F1EDE4"
          strokeWidth="1.5"
          className="dark:fill-[#EDE9DF]"
        />
        {/* Egg Yolk */}
        <circle cx="138" cy="76" r="13" fill="#F59E0B" />
        <circle cx="136" cy="74" r="11" fill="#FBBF24" />
        <circle cx="133" cy="71" r="3.5" fill="#FFFFFF" opacity="0.8" />
      </g>

      {/* Warm Sprinkles / Sparkles */}
      <path
        d="M190 40 L193 48 L185 45 Z"
        fill="var(--brand-accent)"
        opacity="0.8"
      />
      <path
        d="M205 32 L207 42 L200 37 Z"
        fill="var(--brand-accent)"
        opacity="0.8"
      />
      <path
        d="M218 45 L223 52 L215 51 Z"
        fill="var(--brand-accent)"
        opacity="0.8"
      />
    </svg>
  );
}

export function RegisterSaladIllustration(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      viewBox="0 0 460 300"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className="w-full max-w-[380px] h-auto select-none"
      {...props}
    >
      {/* Background Soft Aura */}
      <ellipse cx="230" cy="225" rx="180" ry="42" fill="var(--brand-tint)" opacity="0.65" />
      <ellipse cx="225" cy="220" rx="130" ry="30" fill="var(--bg-subtle)" />

      {/* Decorative Leaves on sides */}
      <path
        d="M55 180 C40 160 40 135 65 125 C80 145 80 168 55 180 Z"
        fill="var(--brand-primary)"
        opacity="0.4"
      />
      <path
        d="M85 205 C75 190 77 175 92 170 C104 182 102 197 85 205 Z"
        fill="var(--brand-primary)"
        opacity="0.4"
      />
      <path
        d="M400 130 C415 110 415 88 395 82 C380 98 380 120 400 130 Z"
        fill="var(--brand-primary)"
        opacity="0.45"
      />

      {/* The Big Nourishing Bowl */}
      <g transform="translate(100, 80)">
        {/* Bowl Shadow */}
        <ellipse cx="130" cy="165" rx="110" ry="22" fill="rgba(0,0,0,0.08)" />

        {/* Bowl Body */}
        <path
          d="M20 70 C20 145 75 168 130 168 C185 168 240 145 240 70 Z"
          fill="#FAF4EB"
          stroke="#D8CEBD"
          strokeWidth="2.5"
          className="dark:fill-[#222924] dark:stroke-[#343e37]"
        />

        {/* Bowl Rim */}
        <ellipse
          cx="130"
          cy="70"
          rx="110"
          ry="30"
          fill="#EDE4D5"
          stroke="#D8CEBD"
          strokeWidth="2.5"
          className="dark:fill-[#1b221d] dark:stroke-[#343e37]"
        />

        {/* Leafy Greens Overflow */}
        <path
          d="M35 62 C25 45 45 32 60 42 C72 28 95 35 92 55 C80 75 48 78 35 62 Z"
          fill="var(--brand-primary)"
        />
        <path
          d="M85 55 C82 36 106 28 122 38 C135 25 158 32 152 52 C142 72 100 75 85 55 Z"
          fill="#3b825e"
        />
        <path
          d="M145 56 C140 38 165 30 180 40 C195 28 215 38 210 58 C202 78 165 78 145 56 Z"
          fill="var(--brand-primary)"
        />

        {/* Avocado Slices */}
        <g transform="translate(60, 52)">
          <path
            d="M0 25 C5 10 25 5 35 15 C45 25 35 45 20 45 C5 45 -5 40 0 25 Z"
            fill="#86EFAC"
            stroke="#15803D"
            strokeWidth="2"
          />
          <circle cx="20" cy="26" r="6" fill="#A3E635" />
        </g>
        <g transform="translate(75, 60)">
          <path
            d="M0 22 C5 8 22 4 32 12 C40 20 32 38 18 38 C5 38 -5 34 0 22 Z"
            fill="#86EFAC"
            stroke="#15803D"
            strokeWidth="2"
          />
          <circle cx="18" cy="22" r="5" fill="#A3E635" />
        </g>

        {/* Protein / Chickpeas / Tofu cubes */}
        <circle cx="115" cy="85" r="9" fill="#FDE047" stroke="#CA8A04" strokeWidth="1.5" />
        <circle cx="132" cy="88" r="8" fill="#FDE047" stroke="#CA8A04" strokeWidth="1.5" />
        <circle cx="125" cy="99" r="8" fill="#FDE047" stroke="#CA8A04" strokeWidth="1.5" />

        {/* Warm Terracotta Grilled Salmon / Pumpkin bites */}
        <rect x="145" y="70" width="22" height="18" rx="5" fill="var(--brand-accent)" />
        <rect x="170" y="75" width="24" height="20" rx="5" fill="var(--brand-accent)" />
        <path d="M148 76 L163 76" stroke="#9A3412" strokeWidth="1.5" strokeLinecap="round" />
        <path d="M174 82 L189 82" stroke="#9A3412" strokeWidth="1.5" strokeLinecap="round" />
      </g>

      {/* Floating Sparkles */}
      <path d="M310 50 L313 58 L305 55 Z" fill="var(--brand-accent)" opacity="0.8" />
      <path d="M325 42 L327 52 L320 47 Z" fill="var(--brand-accent)" opacity="0.8" />
    </svg>
  );
}

export function ForgotPaperPlaneIllustration(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      viewBox="0 0 380 260"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className="w-full max-w-[320px] h-auto select-none"
      {...props}
    >
      {/* Soft Background Cloud */}
      <ellipse cx="190" cy="190" rx="140" ry="32" fill="var(--brand-tint)" opacity="0.6" />

      {/* Flight Dotted Trail */}
      <path
        d="M170 120 C200 90 220 80 240 100 C255 115 250 140 230 135 C215 130 220 105 245 80 L280 45"
        stroke="#D8CEBD"
        strokeWidth="2.5"
        strokeDasharray="5 5"
        strokeLinecap="round"
        className="dark:stroke-[#455047]"
      />

      {/* The Friendly Green Envelope */}
      <g transform="translate(70, 70)">
        {/* Envelope Shadow */}
        <ellipse cx="75" cy="115" rx="65" ry="12" fill="rgba(0,0,0,0.07)" />

        {/* Envelope Body */}
        <rect
          x="10"
          y="25"
          width="130"
          height="85"
          rx="14"
          fill="var(--brand-primary)"
        />

        {/* Envelope Flap Interior Fold Lines */}
        <path
          d="M10 25 L75 75 L140 25"
          fill="#3b825e"
          stroke="var(--brand-primary-hover)"
          strokeWidth="2"
        />

        {/* Envelope Top Open Flap */}
        <path
          d="M10 25 L75 0 L140 25"
          fill="#3b825e"
          stroke="var(--brand-primary-hover)"
          strokeWidth="2"
        />

        {/* Letter Peeking Out */}
        <path
          d="M25 28 L125 28 L125 55 L25 55 Z"
          fill="#FAF8F5"
          rx="4"
          className="dark:fill-[#EDE9DF]"
        />
        <line x1="38" y1="38" x2="112" y2="38" stroke="#D1C7B7" strokeWidth="2.5" strokeLinecap="round" />
        <line x1="38" y1="46" x2="95" y2="46" stroke="#D1C7B7" strokeWidth="2.5" strokeLinecap="round" />
      </g>

      {/* The Paper Airplane Gliding */}
      <g transform="translate(265, 30)">
        <path
          d="M0 35 L48 0 L32 45 L18 36 Z"
          fill="var(--brand-accent)"
        />
        <path
          d="M48 0 L18 36 L12 48 L22 38 Z"
          fill="#C25A23"
        />
        <path
          d="M48 0 L0 35 L18 36 Z"
          fill="#E7844E"
        />
      </g>

      {/* Warm floating leaf sprinkles */}
      <path d="M45 130 C35 120 40 105 52 108 C58 118 55 128 45 130 Z" fill="var(--brand-accent)" opacity="0.6" />
      <path d="M40 150 C32 142 36 132 45 135 C50 142 48 148 40 150 Z" fill="var(--brand-accent)" opacity="0.6" />
    </svg>
  );
}

export function SuccessCheckIllustration({
  className = "",
}: {
  className?: string;
}) {
  return (
    <div className={`flex items-center justify-center relative my-2 ${className}`}>
      <div className="relative">
        {/* Main Soft Green Circle */}
        <div className="w-20 h-20 rounded-full bg-[var(--brand-primary)] text-white flex items-center justify-center shadow-none">
          <svg
            className="w-10 h-10"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="3"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <polyline points="20 6 9 17 4 12" />
          </svg>
        </div>

        {/* Decorative Floating Sprinkles */}
        <svg
          className="absolute -top-3 -right-5 w-8 h-8 select-none pointer-events-none"
          viewBox="0 0 32 32"
          fill="none"
        >
          <path d="M12 4 L16 12 L8 9 Z" fill="var(--brand-accent)" />
          <path d="M22 10 L25 18 L18 16 Z" fill="var(--brand-accent)" />
        </svg>
      </div>
    </div>
  );
}

export function OrganicWaveBackdrop() {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute inset-0 z-0 overflow-hidden select-none opacity-40 dark:opacity-20"
    >
      {/* Bottom organic curve hills */}
      <svg
        className="absolute -bottom-10 left-0 w-full min-w-[1000px] h-64 text-[var(--brand-tint)]"
        viewBox="0 0 1440 320"
        fill="currentColor"
        preserveAspectRatio="none"
      >
        <path d="M0,192L60,176C120,160,240,128,360,138.7C480,149,600,203,720,208C840,213,960,171,1080,149.3C1200,128,1320,128,1380,128L1440,128L1440,320L1380,320C1320,320,1200,320,1080,320C960,320,840,320,720,320C600,320,480,320,360,320C240,320,120,320,60,320L0,320Z" />
      </svg>
      {/* Bottom left botanical leaf silhouette */}
      <svg
        className="absolute -bottom-8 -left-8 w-44 h-44 text-[var(--brand-primary)] opacity-20"
        viewBox="0 0 100 100"
        fill="currentColor"
      >
        <path d="M10 90 C10 40 50 10 90 10 C90 60 50 90 10 90 Z" />
      </svg>
      {/* Top right botanical leaf silhouette */}
      <svg
        className="absolute top-12 -right-10 w-48 h-48 text-[var(--brand-primary)] opacity-15 rotate-45"
        viewBox="0 0 100 100"
        fill="currentColor"
      >
        <path d="M10 90 C10 40 50 10 90 10 C90 60 50 90 10 90 Z" />
      </svg>
    </div>
  );
}
