export function formatIDR(amount: number): string {
  const safe = Number.isFinite(amount) ? Math.max(0, Math.round(amount)) : 0;
  return `Rp${safe.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ".")}`;
}

export function formatMacro(value: number): string {
  if (!Number.isFinite(value)) return "0";
  const rounded = Math.round(Math.max(0, value) * 10) / 10;
  return Number.isInteger(rounded) ? String(rounded) : rounded.toFixed(1);
}
