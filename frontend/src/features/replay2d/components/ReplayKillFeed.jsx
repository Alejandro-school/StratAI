import { Crosshair, Skull } from "lucide-react";

export function ReplayKillFeed({ events, tick, tickRate }) {
  const kills = events
    .filter((event) => event.type === "kill" && event.tick <= tick && tick - event.tick < tickRate * 5)
    .slice(-4)
    .reverse();
  if (!kills.length) return null;
  return (
    <ol className="r2-kill-feed" aria-label="Eliminaciones recientes">
      {kills.map((kill) => (
        <li key={kill.id}>
          <span className={kill.killer_team?.toLowerCase()}>{kill.killer_name || "Jugador"}</span>
          <span className="r2-kill-weapon"><Crosshair size={11} />{kill.weapon || "arma"}{kill.headshot && <Skull size={11} />}</span>
          <span className={kill.victim_team?.toLowerCase()}>{kill.victim_name || "Jugador"}</span>
        </li>
      ))}
    </ol>
  );
}
