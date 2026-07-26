import { Skull } from "lucide-react";
import {
  buildPlayerLoadout,
  describePlayerLoadout,
  equipmentIconPath,
  equipmentLabel,
  playerIdentity,
  stableRosterPlayers,
} from "../domain/weaponPresentation";

const MONEY_FORMAT = new Intl.NumberFormat("en-US", {
  maximumFractionDigits: 0,
  useGrouping: false,
});

const ICON_DIMENSIONS = {
  primary: { width: 78, height: 24 },
  pistol: { width: 40, height: 22 },
  utility: { width: 18, height: 18 },
  badge: { width: 16, height: 16 },
};

const EquipmentIcon = ({ equipment, active = false, size = "utility", className = "" }) => {
  if (!equipment) return <span className={`r2-equipment-placeholder ${size}`} aria-hidden="true" />;
  const source = equipmentIconPath(equipment);
  if (!source) return <span className={`r2-equipment-text ${active ? "active" : ""}`}>{equipment}</span>;
  const dimensions = ICON_DIMENSIONS[size] || ICON_DIMENSIONS.utility;
  return (
    <img
      src={source}
      alt=""
      aria-hidden="true"
      title={equipmentLabel(equipment)}
      width={dimensions.width}
      height={dimensions.height}
      className={`r2-equipment-icon ${size} ${active ? "active" : ""} ${className}`}
    />
  );
};

const PlayerEquipment = ({ player }) => {
  const loadout = buildPlayerLoadout(player);
  const utility = loadout.utility.slice(0, 4);
  return (
    <span className="r2-player-equipment" aria-hidden="true">
      <span className="r2-main-weapons">
        <EquipmentIcon equipment={loadout.primary} active={loadout.primary === loadout.active} size="primary" />
        <EquipmentIcon equipment={loadout.pistol} active={loadout.pistol === loadout.active} size="pistol" />
      </span>
      <span className="r2-utility-icons">
        {Array.from({ length: 4 }, (_, index) => (
          <EquipmentIcon
            key={`${utility[index] || "empty"}-${index}`}
            equipment={utility[index]}
            active={utility[index] === loadout.active}
          />
        ))}
      </span>
    </span>
  );
};

const PlayerVitals = ({ player, health, armor }) => (
  <span className="r2-player-vitals" style={{ "--health": `${health}%`, "--armor": `${armor}%` }}>
    <span className="r2-health-ring">
      {player.alive ? health : <Skull size={15} aria-hidden="true" />}
    </span>
    {armor > 0 && (
      <span className="r2-armor-mark" title={player.has_helmet ? "Casco y kevlar" : "Kevlar"}>
        <EquipmentIcon equipment={player.has_helmet ? "helmet" : "kevlar"} size="badge" />
      </span>
    )}
    {player.has_c4 && (
      <span className="r2-c4-badge" title="Porta la C4">
        <EquipmentIcon equipment="c4" size="badge" />
      </span>
    )}
  </span>
);

const PlayerRow = ({ player, selected, number, onSelect }) => {
  const health = Math.max(0, player.health || 0);
  const armor = Math.max(0, player.armor || 0);
  const money = MONEY_FORMAT.format(Number(player.money || 0));
  const loadoutDescription = describePlayerLoadout(player);
  return (
    <button
      type="button"
      className={`r2-player-row ${player.team.toLowerCase()} ${!player.alive ? "dead" : ""} ${selected ? "selected" : ""}`}
      onClick={() => onSelect(playerIdentity(player))}
      aria-pressed={selected}
      aria-label={`${player.name}, ${health} de vida, ${armor} de armadura, $${money}. ${loadoutDescription}`}
      title={loadoutDescription}
    >
      <PlayerVitals player={player} health={health} armor={armor} />
      <span className="r2-player-identity">
        <strong title={player.name}><i>{number}</i>{player.name || "Jugador"}</strong>
        <small>
          <span className="r2-player-money">${money}</span>
          {player.has_defuse_kit && (
            <span className="r2-kit-badge" title="Kit de desactivación">
              <EquipmentIcon equipment="defuser" size="badge" />
              KIT
            </span>
          )}
        </small>
      </span>
      <PlayerEquipment player={player} />
    </button>
  );
};

const TeamBlock = ({ team, players, score, focusPlayerId, onSelect }) => (
  <section className={`r2-team-block ${team.toLowerCase()}`}>
    <header>
      <div>
        <strong>Team {team === "CT" ? "A" : "B"}</strong>
        <small>{team === "CT" ? "Counter-Terrorists" : "Terrorists"}</small>
      </div>
      <b>{score}</b>
    </header>
    <div className="r2-player-list">
      {players.map((player, index) => (
        <PlayerRow
          key={playerIdentity(player)}
          player={player}
          number={index + 1}
          selected={playerIdentity(player) === String(focusPlayerId)}
          onSelect={onSelect}
        />
      ))}
    </div>
  </section>
);

export function ReplayRosterPanel({ players = [], ctScore, tScore, focusPlayerId, onSelect }) {
  const counterTerrorists = stableRosterPlayers(players, "CT");
  const terrorists = stableRosterPlayers(players, "T");
  return (
    <aside className="r2-roster" aria-label="Jugadores de la ronda">
      <TeamBlock team="CT" players={counterTerrorists} score={ctScore} focusPlayerId={focusPlayerId} onSelect={onSelect} />
      <TeamBlock team="T" players={terrorists} score={tScore} focusPlayerId={focusPlayerId} onSelect={onSelect} />
    </aside>
  );
}
