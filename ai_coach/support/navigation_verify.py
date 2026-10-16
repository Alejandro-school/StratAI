"""Independent Python checks of the Go candidate graph, associations and paths.

This verifies a centre-weighted graph diagnostic, never physical traversability.
"""

import heapq
import math
from itertools import pairwise

import numpy as np

from ai_coach.support.anchors import require


def xyz(p):
    return tuple(p[k] for k in ("X", "Y", "Z"))


def inside(point, polygon):
    x, y, _ = point
    hit = False
    for left, right in zip(polygon, polygon[-1:] + polygon[:-1]):
        ax, ay, _ = left
        bx, by, _ = right
        cross = (y - ay) * (bx - ax) - (x - ax) * (by - ay)
        if (
            abs(cross) <= 1e-9
            and min(ax, bx) - 1e-9 <= x <= max(ax, bx) + 1e-9
            and min(ay, by) - 1e-9 <= y <= max(ay, by) + 1e-9
        ):
            return True
        if (ay > y) != (by > y) and x < (bx - ax) * (y - ay) / (by - ay) + ax:
            hit = not hit
    return hit


class GraphVerifier:
    def __init__(self, graph):
        self.nodes = {n["ID"]: n for n in graph["Nodes"]}
        require(len(self.nodes) == len(graph["Nodes"]) > 0, "duplicate/empty NAV nodes")
        self.ids = sorted(self.nodes)
        self.centres = np.array([xyz(self.nodes[i]["Position"]) for i in self.ids])
        require(np.isfinite(self.centres).all(), "nonfinite NAV centres")
        self.polygons = [[xyz(c) for c in self.nodes[i]["Corners"]] for i in self.ids]
        require(all(len(p) >= 3 for p in self.polygons), "NAV polygon absent")
        require(
            all(np.isfinite(p).all() for p in self.polygons), "nonfinite NAV polygon"
        )
        require(
            np.allclose(
                self.centres,
                [np.mean(p, axis=0) for p in self.polygons],
                rtol=1e-12,
                atol=1e-9,
            ),
            "NAV centre differs from corners",
        )
        self.mins = np.array([np.min(p, axis=0) for p in self.polygons])
        self.maxs = np.array([np.max(p, axis=0) for p in self.polygons])
        self.edges = {}
        self.associations = {}
        self.source_distances = {}
        self.centre_ids = {}
        for i, n in self.nodes.items():
            self.centre_ids.setdefault(xyz(n["Position"]), []).append(i)
        self.adj = {i: [] for i in self.ids}
        for e in graph["Edges"]:
            key = e["FromID"], e["ToID"]
            require(
                key not in self.edges and all(i in self.nodes for i in key),
                "invalid NAV edge",
            )
            w = math.dist(
                xyz(self.nodes[key[0]]["Position"]), xyz(self.nodes[key[1]]["Position"])
            )
            require(
                w > 0 and math.isclose(w, e["Weight"], rel_tol=1e-12, abs_tol=1e-9),
                "NAV edge weight mismatch",
            )
            self.edges[key] = w
            self.adj[key[0]].append((key[1], w))

    def associate(self, point):
        if point not in self.associations:
            self.associations[point] = self._associate(point)
        return self.associations[point]

    def _associate(self, point):
        candidates = np.flatnonzero(
            (self.mins[:, 0] - 1e-9 <= point[0])
            & (point[0] <= self.maxs[:, 0] + 1e-9)
            & (self.mins[:, 1] - 1e-9 <= point[1])
            & (point[1] <= self.maxs[:, 1] + 1e-9)
            & (np.abs(self.centres[:, 2] - point[2]) <= 128)
        )
        contained = [i for i in candidates if inside(point, self.polygons[i])]
        if contained:
            index = min(
                contained,
                key=lambda i: (abs(self.centres[i, 2] - point[2]), self.ids[i]),
            )
            return (
                self.ids[index],
                math.dist(point, self.centres[index]),
                "polygon_xy_candidate",
            )
        distances = np.linalg.norm(self.centres - point, axis=1)
        index = int(np.argmin(distances))
        if distances[index] <= 96:
            return (
                self.ids[index],
                math.dist(point, self.centres[index]),
                "nearest_center_candidate",
            )
        return None, None, "outside_candidate_coverage"

    def distances(self, start):
        if start in self.source_distances:
            return self.source_distances[start]
        # Independent single-source search; Go queries a single target per run.
        distances = {start: 0.0}
        queue = [(0.0, start)]
        while queue:
            value, node = heapq.heappop(queue)
            if value != distances[node]:
                continue
            for target, weight in self.adj[node]:
                candidate = value + weight
                if candidate < distances.get(target, math.inf):
                    distances[target] = candidate
                    heapq.heappush(queue, (candidate, target))
        self.source_distances[start] = distances
        return distances

    def verify(self, query, trace):
        origin, target = xyz(query["origin"]), xyz(query["target"])
        require(
            all(math.isfinite(x) for p in (origin, target) for x in p),
            "nonfinite endpoint",
        )
        a, da, ma = self.associate(origin)
        b, db, mb = self.associate(target)
        m = trace["Measurement"]
        require(
            math.isclose(
                m["DirectDistance"],
                math.dist(origin, target),
                rel_tol=1e-12,
                abs_tol=1e-8,
            ),
            "direct distance mismatch",
        )
        if a is None or b is None:
            reason = (
                "both_endpoints_outside_nav_tolerance"
                if a is b
                else (
                    "origin_outside_nav_tolerance"
                    if a is None
                    else "target_outside_nav_tolerance"
                )
            )
            require(
                m["Reachable"] is None
                and m["PathDistance"] is None
                and m["OriginAssociationDistance"] is None
                and m["TargetAssociationDistance"] is None
                and m["UnavailableReason"] == reason
                and trace["Points"] == [],
                "NAV coverage mismatch",
            )
        else:
            require(
                math.isclose(da, m["OriginAssociationDistance"], abs_tol=1e-8)
                and math.isclose(db, m["TargetAssociationDistance"], abs_tol=1e-8),
                "NAV endpoint offset mismatch",
            )
            distance = self.distances(a).get(b)
            if distance is None:
                require(
                    m["Reachable"] is False
                    and m["PathDistance"] is None
                    and m["UnavailableReason"] == "no_navigable_connection"
                    and trace["Points"] == [],
                    "directed no-path mismatch",
                )
            else:
                require(
                    m["Reachable"] is True
                    and m["UnavailableReason"] == ""
                    and math.isclose(
                        m["PathDistance"],
                        da + distance + db,
                        rel_tol=1e-12,
                        abs_tol=1e-7,
                    ),
                    "shortest NAV distance mismatch",
                )
                points = [xyz(p) for p in trace["Points"]]
                require(
                    points[0] == origin and points[-1] == target,
                    "NAV trace endpoints mismatch",
                )
                # A route must actually follow directed edges; length alone is insufficient.
                chain = [xyz(self.nodes[a]["Position"])]
                chain += points[1:-1]
                chain += [xyz(self.nodes[b]["Position"])]
                unique = [p for i, p in enumerate(chain) if i == 0 or p != chain[i - 1]]
                possible = {a}
                for pos in unique[1:]:
                    possible = {
                        j
                        for j in self.centre_ids.get(pos, [])
                        if any((i, j) in self.edges for i in possible)
                    }
                require(b in possible, "NAV trace has unsupported directed segment")
                require(
                    math.isclose(
                        sum(math.dist(x, y) for x, y in pairwise(points)),
                        m["PathDistance"],
                        rel_tol=1e-12,
                        abs_tol=1e-7,
                    ),
                    "NAV polyline length mismatch",
                )
        return {"origin": ma, "target": mb, "same_polygon": a is not None and a == b}
