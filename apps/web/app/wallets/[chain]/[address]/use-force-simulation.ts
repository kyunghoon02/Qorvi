import type { Edge, Node } from "@xyflow/react";
import { useEffect, useRef } from "react";

interface PhysicsNode {
  x: number;
  y: number;
  vx: number;
  vy: number;
  dragging: boolean;
}

export function useForceSimulation<T extends Node>(
  nodes: T[],
  setNodes: React.Dispatch<React.SetStateAction<T[]>>,
  edges: Edge[],
  primaryNodeId?: string,
) {
  const physicsRef = useRef<Map<string, PhysicsNode>>(new Map());

  // Sync positions from ReactFlow to physics engine (e.g. for initial load and dragging)
  const map = physicsRef.current;
  for (const n of nodes) {
    if (!map.has(n.id)) {
      map.set(n.id, {
        x: n.position.x || 0,
        y: n.position.y || 0,
        vx: (Math.random() - 0.5) * 20,
        vy: (Math.random() - 0.5) * 20,
        dragging: !!n.dragging,
      });
    } else {
      const p = map.get(n.id);
      if (!p) {
        continue;
      }
      p.dragging = !!n.dragging;
      if (p.dragging) {
        // If user is dragging node, override physics position
        p.x = n.position.x;
        p.y = n.position.y;
        p.vx = 0;
        p.vy = 0;
      }
    }
  }

  useEffect(() => {
    let animationId: number;

    const simulate = () => {
      let isMoving = false;
      const simNodes = Array.from(physicsRef.current.entries());

      const REPULSION = 200000;
      const SPRING_LENGTH = 350;
      const K_SPRING = 0.08;
      const DAMPING = 0.65;
      const CENTER_FORCE = 0.03;

      const centerX = 590;
      const centerY = 450;

      // 1. Repulsion
      for (let i = 0; i < simNodes.length; i++) {
        for (let j = i + 1; j < simNodes.length; j++) {
          const pairA = simNodes[i];
          const pairB = simNodes[j];
          if (!pairA || !pairB) {
            continue;
          }
          const [, a] = pairA;
          const [, b] = pairB;
          if (a.dragging && b.dragging) continue;

          const dx = a.x - b.x;
          const dy = a.y - b.y;
          let distSq = dx * dx + dy * dy;
          if (distSq === 0) distSq = 1;

          if (distSq < 1500000) {
            const f = REPULSION / distSq;
            const dist = Math.sqrt(distSq);
            if (!a.dragging) {
              a.vx += (dx / dist) * f;
              a.vy += (dy / dist) * f;
            }
            if (!b.dragging) {
              b.vx -= (dx / dist) * f;
              b.vy -= (dy / dist) * f;
            }
          }
        }
      }

      // 2. Spring Forces
      for (const edge of edges) {
        const source = physicsRef.current.get(edge.source);
        const target = physicsRef.current.get(edge.target);
        if (source && target) {
          const dx = target.x - source.x;
          const dy = target.y - source.y;
          const dist = Math.sqrt(dx * dx + dy * dy) || 1;
          const f = (dist - SPRING_LENGTH) * K_SPRING;

          if (!source.dragging) {
            source.vx += (dx / dist) * f;
            source.vy += (dy / dist) * f;
          }
          if (!target.dragging) {
            target.vx -= (dx / dist) * f;
            target.vy -= (dy / dist) * f;
          }
        }
      }

      // 3. Integration & Center Force
      for (const [id, p] of simNodes) {
        if (!p.dragging) {
          if (id === primaryNodeId) {
            // Pull primary node strongly to center
            p.x += (centerX - p.x) * 0.15;
            p.y += (centerY - p.y) * 0.15;
            p.vx *= 0.1;
            p.vy *= 0.1;
          } else {
            p.vx -= (p.x - centerX) * CENTER_FORCE;
            p.vy -= (p.y - centerY) * CENTER_FORCE;
            p.vx *= DAMPING;
            p.vy *= DAMPING;
            p.x += p.vx;
            p.y += p.vy;
          }

          if (Math.abs(p.vx) > 0.5 || Math.abs(p.vy) > 0.5) {
            isMoving = true;
          }
        }
      }

      // 4. State Update (Render)
      if (isMoving) {
        setNodes((nds) =>
          nds.map((n) => {
            const p = physicsRef.current.get(n.id);
            if (p && !n.dragging) {
              // Ensure we return a new position object so ReactFlow detects changes
              return { ...n, position: { x: p.x, y: p.y } };
            }
            return n;
          }),
        );
      }

      animationId = requestAnimationFrame(simulate);
    };

    animationId = requestAnimationFrame(simulate);

    return () => cancelAnimationFrame(animationId);
  }, [edges, primaryNodeId, setNodes]);
}
