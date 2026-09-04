import type { AvatarKey, RoutineIconKey } from "./types";

import capybara from "./assets/illustrations/avatars/capybara.webp";
import cat from "./assets/illustrations/avatars/cat.webp";
import chick from "./assets/illustrations/avatars/chick.webp";
import dog from "./assets/illustrations/avatars/dog.webp";
import fox from "./assets/illustrations/avatars/fox.webp";
import lion from "./assets/illustrations/avatars/lion.webp";
import owl from "./assets/illustrations/avatars/owl.webp";
import panda from "./assets/illustrations/avatars/panda.webp";
import penguin from "./assets/illustrations/avatars/penguin.webp";
import rabbit from "./assets/illustrations/avatars/rabbit.webp";
import sheep from "./assets/illustrations/avatars/sheep.webp";
import tiger from "./assets/illustrations/avatars/tiger.webp";
import bath from "./assets/illustrations/routines/bath.webp";
import chores from "./assets/illustrations/routines/chores.webp";
import exercise from "./assets/illustrations/routines/exercise.webp";
import family from "./assets/illustrations/routines/family.webp";
import general from "./assets/illustrations/routines/general.webp";
import meal from "./assets/illustrations/routines/meal.webp";
import music from "./assets/illustrations/routines/music.webp";
import outdoor from "./assets/illustrations/routines/outdoor.webp";
import reading from "./assets/illustrations/routines/reading.webp";
import school from "./assets/illustrations/routines/school.webp";
import sleep from "./assets/illustrations/routines/sleep.webp";
import study from "./assets/illustrations/routines/study.webp";

export const avatars: { key: AvatarKey; label: string; src: string }[] = [
  { key: "capybara", label: "Capivara", src: capybara }, { key: "cat", label: "Gato", src: cat },
  { key: "chick", label: "Pintinho", src: chick }, { key: "dog", label: "Cachorro", src: dog },
  { key: "fox", label: "Raposa", src: fox },
  { key: "lion", label: "Leão", src: lion }, { key: "owl", label: "Coruja", src: owl },
  { key: "panda", label: "Panda", src: panda }, { key: "penguin", label: "Pinguim", src: penguin },
  { key: "rabbit", label: "Coelho", src: rabbit }, { key: "sheep", label: "Ovelha", src: sheep },
  { key: "tiger", label: "Tigre", src: tiger },
];

export const routineIcons: { key: RoutineIconKey; label: string; src: string }[] = [
  { key: "study", label: "Estudo", src: study }, { key: "reading", label: "Leitura", src: reading },
  { key: "sleep", label: "Dormir", src: sleep }, { key: "bath", label: "Banho", src: bath },
  { key: "meal", label: "Refeição", src: meal }, { key: "school", label: "Escola", src: school },
  { key: "exercise", label: "Atividade física", src: exercise }, { key: "chores", label: "Tarefas da casa", src: chores },
  { key: "family", label: "Família", src: family }, { key: "music", label: "Música", src: music },
  { key: "outdoor", label: "Ao ar livre", src: outdoor }, { key: "general", label: "Outra rotina", src: general },
];

const avatarByKey = new Map(avatars.map((item) => [item.key, item]));
const legacyAvatarAliases: Record<string, AvatarKey> = {
  cat_bow: "cat", rabbit_flower: "rabbit", panda_flower: "panda", fox_bow: "fox",
};
const routineByKey = new Map(routineIcons.map((item) => [item.key, item]));

export function isAvatarKey(value: unknown): value is AvatarKey { return typeof value === "string" && avatarByKey.has(value as AvatarKey); }
export function isRoutineIconKey(value: unknown): value is RoutineIconKey { return typeof value === "string" && routineByKey.has(value as RoutineIconKey); }

export function defaultAvatarKey(id: string): AvatarKey {
  let hash = 0;
  for (const character of id) hash = (hash * 31 + character.charCodeAt(0)) >>> 0;
  return avatars[hash % avatars.length].key;
}

export function normalizeAvatarKey(value: unknown, deviceID: string): AvatarKey {
  if (isAvatarKey(value)) return value;
  if (typeof value === "string" && legacyAvatarAliases[value]) return legacyAvatarAliases[value];
  return defaultAvatarKey(deviceID);
}

export function inferRoutineIcon(name: string): RoutineIconKey {
  const normalized = name.normalize("NFD").replace(/[\u0300-\u036f]/g, "").toLowerCase();
  if (/leit|livro/.test(normalized)) return "reading";
  if (/dorm|sono|descans/.test(normalized)) return "sleep";
  if (/banh|chuveir/.test(normalized)) return "bath";
  if (/almoc|jantar|cafe|refeic|comer|lanche/.test(normalized)) return "meal";
  if (/escola|aula/.test(normalized)) return "school";
  if (/exerc|esport|futebol|atividade fisica/.test(normalized)) return "exercise";
  if (/arrum|limp|tarefa de casa|organiz/.test(normalized)) return "chores";
  if (/famil/.test(normalized)) return "family";
  if (/music|instrument/.test(normalized)) return "music";
  if (/passe|parque|ar livre|brinc/.test(normalized)) return "outdoor";
  if (/estud|dever|licao|tarefa/.test(normalized)) return "study";
  return "general";
}

export function DeviceAvatar({ avatarKey, name, className = "" }: { avatarKey: AvatarKey; name: string; className?: string }) {
  const avatar = avatarByKey.get(avatarKey) ?? avatars[0];
  return <img alt={`Avatar ${avatar.label.toLowerCase()} de ${name}`} className={`device-avatar ${className}`} src={avatar.src} />;
}

export function RoutineVisual({ iconKey, className = "" }: { iconKey: RoutineIconKey; className?: string }) {
  const icon = routineByKey.get(iconKey) ?? routineIcons.at(-1)!;
  return <img alt="" aria-hidden="true" className={`routine-visual ${className}`} src={icon.src} />;
}

export function AvatarPicker({ value, onChange }: { value: AvatarKey; onChange: (value: AvatarKey) => void }) {
  return <fieldset className="visual-picker avatar-picker"><legend>Avatar do computador</legend><div>{avatars.map((avatar) => <button aria-label={avatar.label} aria-pressed={value === avatar.key} className={value === avatar.key ? "active" : ""} key={avatar.key} onClick={() => onChange(avatar.key)} type="button"><img alt="" src={avatar.src} /><span>{avatar.label}</span></button>)}</div></fieldset>;
}

export function RoutineIconPicker({ value, onChange }: { value: RoutineIconKey; onChange: (value: RoutineIconKey) => void }) {
  return <fieldset className="visual-picker routine-icon-picker"><legend>Desenho da rotina</legend><div>{routineIcons.map((icon) => <button aria-label={icon.label} aria-pressed={value === icon.key} className={value === icon.key ? "active" : ""} key={icon.key} onClick={() => onChange(icon.key)} type="button"><img alt="" src={icon.src} /><span>{icon.label}</span></button>)}</div></fieldset>;
}
