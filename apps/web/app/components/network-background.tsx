"use client";

import { Canvas, useFrame } from "@react-three/fiber";
import { useEffect, useMemo, useRef, useState } from "react";
import * as THREE from "three";

function ParticleNetwork() {
  const pointsRef = useRef<THREE.Points | null>(null);
  const linesRef = useRef<THREE.LineSegments | null>(null);

  const { pointsGeometry, linesGeometry, hasLines } = useMemo(() => {
    const pArray: number[] = [];
    const cArray: number[] = [];
    const linePositions: number[] = [];
    const lineColors: number[] = [];

    const rootCount = 15;
    const maxSteps = 80;

    const palette = [
      0x00f0ff, // neon blue
      0xff00cc, // hot pink
      0x00ff66, // neon green
      0xffeb3b, // yellow
      0xff3366, // red-pink
      0x8a2be2, // blue-violet
    ];

    interface Strand {
      pos: THREE.Vector3;
      dir: THREE.Vector3;
      step: number;
      color: THREE.Color;
      isMain: boolean;
    }

    const queue: Strand[] = [];

    for (let i = 0; i < rootCount; i++) {
      // Create roots near center
      const startPos = new THREE.Vector3(
        (Math.random() - 0.5) * 4,
        (Math.random() - 0.5) * 4,
        (Math.random() - 0.5) * 4 - 2, // Offset backwards slightly less
      );
      const baseDir = new THREE.Vector3(
        Math.random() - 0.5,
        Math.random() - 0.5,
        Math.random() - 0.5,
      ).normalize();

      const coreColor = new THREE.Color(
        palette[Math.floor(Math.random() * palette.length)],
      );

      // Root points
      pArray.push(startPos.x, startPos.y, startPos.z);
      cArray.push(coreColor.r, coreColor.g, coreColor.b);

      const strandsNum = 3 + Math.random() * 4;
      for (let s = 0; s < strandsNum; s++) {
        const dir = baseDir.clone();
        dir.x += (Math.random() - 0.5) * 0.3;
        dir.y += (Math.random() - 0.5) * 0.3;
        dir.z += (Math.random() - 0.5) * 0.3;
        dir.normalize();

        const col = coreColor
          .clone()
          .offsetHSL((Math.random() - 0.5) * 0.1, 0, 0);
        queue.push({
          pos: startPos.clone(),
          dir,
          step: 0,
          color: col,
          isMain: true,
        });
      }
    }

    while (queue.length > 0) {
      const strand = queue.shift();
      if (!strand) {
        continue;
      }
      const { pos, dir, step, color, isMain } = strand;
      if (step > maxSteps) continue;

      const length = 0.3 + Math.random() * 0.3;
      const nextPos = pos.clone().add(dir.clone().multiplyScalar(length));

      if (Math.random() > 0.8 || step === maxSteps) {
        pArray.push(nextPos.x, nextPos.y, nextPos.z);
        cArray.push(color.r, color.g, color.b);
      }

      linePositions.push(pos.x, pos.y, pos.z, nextPos.x, nextPos.y, nextPos.z);
      lineColors.push(color.r, color.g, color.b, color.r, color.g, color.b);

      // smooth curve
      const nextDir = dir.clone();
      const wander = isMain ? 0.15 : 0.3;
      nextDir.x += (Math.random() - 0.5) * wander;
      nextDir.y += (Math.random() - 0.5) * wander;
      nextDir.z += (Math.random() - 0.5) * wander;
      nextDir.normalize();

      // branch?
      if (step > 10 && step < maxSteps - 10 && Math.random() < 0.05) {
        const branchDir = nextDir.clone();
        branchDir.x += (Math.random() - 0.5) * 0.8;
        branchDir.y += (Math.random() - 0.5) * 0.8;
        branchDir.z += (Math.random() - 0.5) * 0.8;
        branchDir.normalize();

        const branchColor = color
          .clone()
          .offsetHSL((Math.random() - 0.5) * 0.15, 0, 0);
        queue.push({
          pos: nextPos.clone(),
          dir: branchDir,
          step: step + 1,
          color: branchColor,
          isMain: false,
        });
      }

      queue.push({
        pos: nextPos.clone(),
        dir: nextDir,
        step: step + 1,
        color,
        isMain,
      });
    }

    const pGeo = new THREE.BufferGeometry();
    pGeo.setAttribute(
      "position",
      new THREE.BufferAttribute(new Float32Array(pArray), 3),
    );
    pGeo.setAttribute(
      "color",
      new THREE.BufferAttribute(new Float32Array(cArray), 3),
    );

    const lGeo = new THREE.BufferGeometry();
    if (linePositions.length > 0) {
      lGeo.setAttribute(
        "position",
        new THREE.BufferAttribute(new Float32Array(linePositions), 3),
      );
      lGeo.setAttribute(
        "color",
        new THREE.BufferAttribute(new Float32Array(lineColors), 3),
      );
    }

    return {
      pointsGeometry: pGeo,
      linesGeometry: lGeo,
      hasLines: linePositions.length > 0,
    };
  }, []);

  useFrame((state) => {
    const t = state.clock.getElapsedTime() * 0.03;
    if (pointsRef.current) {
      pointsRef.current.rotation.y = t;
      pointsRef.current.rotation.x = t * 0.4;
    }
    if (linesRef.current) {
      linesRef.current.rotation.y = t;
      linesRef.current.rotation.x = t * 0.4;
    }
  });

  return (
    <group>
      <points ref={pointsRef} geometry={pointsGeometry}>
        <pointsMaterial
          transparent
          vertexColors
          size={0.3}
          sizeAttenuation={true}
          depthWrite={false}
          blending={THREE.AdditiveBlending}
          onBeforeCompile={(shader) => {
            shader.fragmentShader = shader.fragmentShader.replace(
              "#include <map_particle_fragment>",
              `
              vec2 xy = gl_PointCoord.xy - vec2(0.5);
              float ll = length(xy);
              if (ll > 0.5) discard;

              float alpha = smoothstep(0.5, 0.0, ll);
              float core = smoothstep(0.15, 0.0, ll);
              diffuseColor.rgb = mix(diffuseColor.rgb, vec3(1.0), core);
              diffuseColor.a *= alpha * 0.9;
              `,
            );
          }}
        />
      </points>
      {hasLines && (
        <lineSegments ref={linesRef} geometry={linesGeometry}>
          <lineBasicMaterial
            vertexColors
            transparent
            opacity={0.35}
            blending={THREE.AdditiveBlending}
            depthWrite={false}
          />
        </lineSegments>
      )}
    </group>
  );
}

function supportsWebGL(): boolean {
  if (typeof document === "undefined") {
    return false;
  }

  try {
    const canvas = document.createElement("canvas");
    return Boolean(
      canvas.getContext("webgl2") ||
        canvas.getContext("webgl") ||
        canvas.getContext("experimental-webgl"),
    );
  } catch {
    return false;
  }
}

export function NetworkBackground() {
  const [canRenderCanvas, setCanRenderCanvas] = useState(false);

  useEffect(() => {
    setCanRenderCanvas(supportsWebGL());
  }, []);

  return (
    <div
      style={{
        position: "absolute",
        inset: 0,
        zIndex: 0,
        background: "var(--bg)",
        pointerEvents: "none",
      }}
    >
      {canRenderCanvas ? (
        <Canvas camera={{ position: [0, 0, 18], fov: 60 }}>
          <fog attach="fog" args={["#121212", 10, 28]} />
          <ParticleNetwork />
        </Canvas>
      ) : null}
    </div>
  );
}
