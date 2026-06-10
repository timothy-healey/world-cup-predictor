// Maps FIFA 3-letter codes to flag emoji.
// Coverage: the 48 confirmed 2026 World Cup nations plus a handful of common
// fallbacks (CRC, etc.). Unknown codes fall back to a generic globe.
const FLAGS: Record<string, string> = {
  ALG: "🇩🇿", ARG: "🇦🇷", AUS: "🇦🇺", AUT: "🇦🇹", BEL: "🇧🇪",
  BIH: "🇧🇦", BRA: "🇧🇷", CAN: "🇨🇦", CIV: "🇨🇮", CMR: "🇨🇲",
  COD: "🇨🇩", COL: "🇨🇴", CPV: "🇨🇻", CRC: "🇨🇷", CRO: "🇭🇷",
  CUW: "🇨🇼", CUR: "🇨🇼", CZE: "🇨🇿", DEN: "🇩🇰", ECU: "🇪🇨",
  EGY: "🇪🇬", ENG: "🏴󠁧󠁢󠁥󠁮󠁧󠁿", ESP: "🇪🇸", FRA: "🇫🇷", GER: "🇩🇪",
  GHA: "🇬🇭", HAI: "🇭🇹", HON: "🇭🇳", IRN: "🇮🇷", IRQ: "🇮🇶",
  ISL: "🇮🇸", ITA: "🇮🇹", JAM: "🇯🇲", JOR: "🇯🇴", JPN: "🇯🇵",
  KOR: "🇰🇷", MAR: "🇲🇦", MEX: "🇲🇽", NED: "🇳🇱", NGA: "🇳🇬",
  NOR: "🇳🇴", NZL: "🇳🇿", PAN: "🇵🇦", PAR: "🇵🇾", PER: "🇵🇪",
  POL: "🇵🇱", POR: "🇵🇹", QAT: "🇶🇦", RSA: "🇿🇦", SAU: "🇸🇦",
  SCO: "🏴󠁧󠁢󠁳󠁣󠁴󠁿", SEN: "🇸🇳", SLV: "🇸🇻", SRB: "🇷🇸", SUI: "🇨🇭",
  SWE: "🇸🇪", TUN: "🇹🇳", TUR: "🇹🇷", UAE: "🇦🇪", UKR: "🇺🇦",
  URU: "🇺🇾", USA: "🇺🇸", UZB: "🇺🇿", VEN: "🇻🇪", WAL: "🏴󠁧󠁢󠁷󠁬󠁳󠁿",
};

export function flagFor(code: string): string {
  return FLAGS[code.toUpperCase()] ?? "🌐";
}
