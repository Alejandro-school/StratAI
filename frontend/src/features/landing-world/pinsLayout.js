const clamp = (value, min, max) => Math.min(Math.max(value, min), max);
const overlaps = (a, b) => Math.abs(a.x - b.x) < 174 && Math.abs(a.y - b.y) < 57;
// Includes the complete guidance, controls and breathing room above/beside it.
export const cameraControlsExclusion = (width, height) => ({ left: 0, right: width <= 760 ? width : width >= 1900 ? 468 : 452, top: height - 270 });

// Layout for this landing's five world callouts. Leaders retain the real anchors.
export function layoutWorldPins(points, width, height) {
  const controls = cameraControlsExclusion(width, height);
  if (width <= 760) {
    const left = width * .25, right = width * .75;
    const lowerRow = Math.min(455, height - 355);
    const coachRow = Math.min(315, lowerRow - 62);
    const slots = { coach: [left, coachRow], utility: [right, Math.min(245, coachRow - 65)], stats: [left, lowerRow], replay: [right, lowerRow], challenges: [width / 2, Math.min(530, height - 307)] };
    return points.map(point => ({ ...point, x: slots[point.id][0], y: slots[point.id][1] }));
  }
  const placed = [];
  for (const point of [...points].sort((a, b) => a.y - b.y)) {
    let destination;
    // The stairs beacon is above the T: leave room for the full posed body.
    const preferredDrop = point.id === 'coach' ? 130 : 65;
    for (const [dx, dy] of [[0, preferredDrop], [-185, preferredDrop], [185, preferredDrop], [0, 130], [0, 0], [-185, 0], [185, 0], [0, -65], [0, -130], [-185, 130], [185, 130], [0, 195], [0, 260]]) {
      const x = clamp(point.x + dx, 100, width - 100);
      const y = clamp(point.y + dy, x < 440 ? 285 : 145, x - 85 < controls.right ? controls.top - 23 : height - 170);
      const candidate = { ...point, x, y };
      if (!placed.some(other => overlaps(candidate, other))) { destination = candidate; break; }
    }
    if (!destination) {
      for (let x = width - 100; x >= 100 && !destination; x -= 185) for (let y = 145; y <= height - 170; y += 65) {
        if (x < 440 && y < 285 || x - 85 < controls.right && y + 23 > controls.top) continue;
        const candidate = { ...point, x, y };
        if (!placed.some(other => overlaps(candidate, other))) { destination = candidate; break; }
      }
    }
    placed.push(destination || { ...point, x: width - 100, y: 145 });
  }
  return placed;
}
