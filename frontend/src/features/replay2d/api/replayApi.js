import { API_URL } from "../../../utils/api";
import { normalizeRound } from "../domain/replayModel";

async function request(path, options) {
  const headers = options?.body
    ? { "Content-Type": "application/json", ...options?.headers }
    : options?.headers;
  const response = await fetch(`${API_URL}${path}`, {
    credentials: "include",
    ...(headers ? { headers } : {}),
    ...options,
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.detail || `Error ${response.status}`);
  }
  if (response.status === 204) return null;
  return response.json();
}

export const replayApi = {
  metadata: (matchId, signal) => request(`/match/${encodeURIComponent(matchId)}/replay/metadata`, { signal }),
  round: async (matchId, round, signal) => normalizeRound(
    await request(`/match/${encodeURIComponent(matchId)}/replay/round/${round}`, { signal }),
  ),
  annotations: (matchId, signal) => request(`/match/${encodeURIComponent(matchId)}/replay/annotations`, { signal }),
  createAnnotation: (matchId, annotation) => request(`/match/${encodeURIComponent(matchId)}/replay/annotations`, {
    method: "POST",
    body: JSON.stringify(annotation),
  }),
  updateAnnotation: (matchId, annotationId, patch) => request(
    `/match/${encodeURIComponent(matchId)}/replay/annotations/${encodeURIComponent(annotationId)}`,
    { method: "PATCH", body: JSON.stringify(patch) },
  ),
  deleteAnnotation: (matchId, annotationId) => request(
    `/match/${encodeURIComponent(matchId)}/replay/annotations/${encodeURIComponent(annotationId)}`,
    { method: "DELETE" },
  ),
};
