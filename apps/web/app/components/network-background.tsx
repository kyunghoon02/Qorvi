"use client";

import { Canvas, useFrame } from "@react-three/fiber";
import { useEffect, useMemo, useRef, useState } from "react";
import * as THREE from "three";

function ParticleNetwork({ count = 400 }) {
  const pointsRef = useRef<THREE.Points | null>(null);
  const linesRef = useRef<THREE.LineSegments | null>(null);

  const { pointsGeometry, linesGeometry, hasLines } = useMemo(() => {
    const pArray = new Float32Array(count * 3);
    const cArray = new Float32Array(count * 3);
    const boxSize = 25;

    // Create random points
    for (let i = 0; i < count; i++) {
      pArray[i * 3] = (Math.random() - 0.5) * boxSize;
      pArray[i * 3 + 1] = (Math.random() - 0.5) * boxSize;
      pArray[i * 3 + 2] = (Math.random() - 0.5) * boxSize - 5;

      // Color nodes based on elegant dark colors
      const colorRandom = Math.random();
      const color = new THREE.Color();
      if (colorRandom > 0.85) {
        color.setHex(0x14b8a6); // subtle teal
      } else if (colorRandom > 0.7) {
        color.setHex(0x8b5cf6); // subtle violet
      } else {
        color.setHex(0x555555); // muted gray
      }
      color.toArray(cArray, i * 3);
    }

    // Connect close points
    const linePositions: number[] = [];
    const maxConnectionDistance = 3.6;

    for (let i = 0; i < count; i++) {
      for (let j = i + 1; j < count; j++) {
        const leftX = pArray[i * 3] ?? 0;
        const leftY = pArray[i * 3 + 1] ?? 0;
        const leftZ = pArray[i * 3 + 2] ?? 0;
        const rightX = pArray[j * 3] ?? 0;
        const rightY = pArray[j * 3 + 1] ?? 0;
        const rightZ = pArray[j * 3 + 2] ?? 0;
        const dx = leftX - rightX;
        const dy = leftY - rightY;
        const dz = leftZ - rightZ;
        const distSq = dx * dx + dy * dy + dz * dz;

        if (distSq < maxConnectionDistance * maxConnectionDistance) {
          linePositions.push(leftX, leftY, leftZ, rightX, rightY, rightZ);
        }
      }
    }

    const pGeo = new THREE.BufferGeometry();
    pGeo.setAttribute("position", new THREE.BufferAttribute(pArray, 3));
    pGeo.setAttribute("color", new THREE.BufferAttribute(cArray, 3));

    const lGeo = new THREE.BufferGeometry();
    if (linePositions.length > 0) {
      lGeo.setAttribute(
        "position",
        new THREE.BufferAttribute(new Float32Array(linePositions), 3),
      );
    }

    return {
      pointsGeometry: pGeo,
      linesGeometry: lGeo,
      hasLines: linePositions.length > 0,
    };
  }, [count]);

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
          size={0.12}
          sizeAttenuation={true}
          depthWrite={false}
          blending={THREE.AdditiveBlending}
        />
      </points>
      {hasLines && (
        <lineSegments ref={linesRef} geometry={linesGeometry}>
          <lineBasicMaterial
            color="#3a3a3a"
            transparent
            opacity={0.25}
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
          <ParticleNetwork count={450} />
        </Canvas>
      ) : null}
    </div>
  );
}
