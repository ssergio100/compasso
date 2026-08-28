export function formatDuration(seconds: number) {
  const total = Math.max(0, Math.round(seconds / 60));
  const hours = Math.floor(total / 60);
  const minutes = total % 60;
  if (!hours) return `${minutes}min`;
  if (!minutes) return `${hours}h`;
  return `${hours}h ${String(minutes).padStart(2, "0")}min`;
}

export function lastSeen(value: string | null) {
  if (!value) return "Ainda não sincronizado";
  const minutes = Math.round((Date.now() - new Date(value).getTime()) / 60000);
  return minutes < 2 ? "Agora mesmo" : `Há ${minutes} minutos`;
}
