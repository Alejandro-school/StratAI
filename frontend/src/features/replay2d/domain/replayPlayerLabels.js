// Presentation only: source participant IDs, observations and model replies stay intact.
export function createTeamPlayerLabels(roster = []) {
  const labels = {};
  for (const side of ["T", "CT"]) {
    const ids = [...new Set(roster.filter((player) => player.team === side)
      .map((player) => String(player.steam_id)))].sort((a, b) => a.localeCompare(b, "en", { numeric: true }));
    ids.forEach((id, index) => { labels[id] = `${side}${index + 1}`; });
  }
  return labels;
}

export function labelCoachText(text, labels) {
  if (!labels || typeof text !== "string") return text;
  return text.replace(/\bP(\d{2,3})\b/g, (original, id) => labels[String(Number(id))] || original);
}

export function labelReplayFrame(frame, labels) {
  if (!frame || !labels) return frame;
  return { ...frame,
    players: (frame.players || []).map((player) => {
      const label = labels[String(player.steam_id)];
      return label ? { ...player, name: label, display_number: Number(label.match(/\d+$/)[0]) } : player;
    }),
    death_markers: (frame.death_markers || []).map((death) => ({ ...death, name: labels[String(death.victim_id)] || death.name })),
    ...(frame.last_seen_markers ? { last_seen_markers: frame.last_seen_markers.map(marker => ({ ...marker,
      name: labels[String(marker.subject_id)] || marker.name })) } : {}),
  };
}

export function labelReplayEvents(events, labels) {
  if (!labels) return events;
  return events.map((event) => {
    const row = { ...event };
    for (const role of ["victim", "killer", "assister", "player", "actor"]) {
      if (labels[String(event[`${role}_id`])]) row[`${role}_name`] = labels[String(event[`${role}_id`])];
    }
    return row;
  });
}
