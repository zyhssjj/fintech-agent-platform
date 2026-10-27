import React, { useState, useEffect, useRef } from "react";

interface SparklineProps {
  data: number[];
  labels: string[];
  width?: number;
  height?: number;
  color?: string;
  prefix?: string;
  suffix?: string;
}

export default function Sparkline({
  data,
  labels,
  width = 110,
  height = 42,
  color = "#D4B88F",
  prefix = "",
  suffix = ""
}: SparklineProps) {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
  const [isMounted, setIsMounted] = useState(false);
  const pathRef = useRef<SVGPathElement>(null);
  const [pathLength, setPathLength] = useState(0);

  useEffect(() => {
    setIsMounted(true);
    if (pathRef.current) {
      const length = pathRef.current.getTotalLength();
      setPathLength(length);
    }
  }, [data]);

  const maxVal = Math.max(...data) * 1.05;
  const minVal = Math.min(...data) * 0.95;
  const valRange = maxVal - minVal || 1;

  // Calculate coordinates
  const points = data.map((val, i) => {
    const x = (i / (data.length - 1)) * (width - 10) + 5;
    const y = height - ((val - minVal) / valRange) * (height - 10) - 5;
    return { x, y, val, label: labels[i] };
  });

  // Build path d attribute
  let pathD = "";
  if (points.length > 0) {
    pathD = `M ${points[0].x} ${points[0].y}`;
    for (let i = 1; i < points.length; i++) {
      // Smooth curve calculation (bezier)
      const prev = points[i - 1];
      const curr = points[i];
      const cpX1 = prev.x + (curr.x - prev.x) / 2;
      const cpY1 = prev.y;
      const cpX2 = prev.x + (curr.x - prev.x) / 2;
      const cpY2 = curr.y;
      pathD += ` C ${cpX1} ${cpY1}, ${cpX2} ${cpY2}, ${curr.x} ${curr.y}`;
    }
  }

  return (
    <div className="relative inline-block select-none" style={{ width, height }}>
      <svg 
        width={width} 
        height={height} 
        className="overflow-visible"
        onMouseLeave={() => setHoveredIndex(null)}
      >
        {/* Draw path with stroke-dasharray animation */}
        <path
          ref={pathRef}
          d={pathD}
          fill="none"
          stroke={color}
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          style={{
            strokeDasharray: pathLength || 1000,
            strokeDashoffset: isMounted ? 0 : pathLength || 1000,
            transition: "stroke-dashoffset 1.5s cubic-bezier(0.25, 1, 0.5, 1)"
          }}
        />

        {/* Interactive hover hotzones and guide circles */}
        {points.map((pt, idx) => (
          <g key={idx}>
            {/* Transparent hotzone circle for easy hover */}
            <circle
              cx={pt.x}
              cy={pt.y}
              r="8"
              fill="transparent"
              className="cursor-pointer"
              onMouseEnter={() => setHoveredIndex(idx)}
            />
            {/* Active highlight circle */}
            {hoveredIndex === idx && (
              <circle
                cx={pt.x}
                cy={pt.y}
                r="4"
                fill={color}
                stroke="#FFFFFF"
                strokeWidth="1.5"
                className="transition-all duration-150 shadow-md"
              />
            )}
          </g>
        ))}
      </svg>

      {/* Floating Micro-Tooltip */}
      {hoveredIndex !== null && points[hoveredIndex] && (
        <div 
          className="absolute z-30 bg-[#222222] text-[#FFFFFF] text-[0.625rem] font-mono py-1 px-1.5 rounded border border-[#D4B88F]/40 shadow-xl pointer-events-none transition-all duration-150 animate-fade-in"
          style={{
            left: `${Math.max(5, Math.min(width - 75, points[hoveredIndex].x - 35))}px`,
            top: `${Math.max(-28, points[hoveredIndex].y - 28)}px`
          }}
        >
          <span className="text-[#999999] mr-1">{points[hoveredIndex].label}:</span>
          <span className="text-[#D4B88F] font-bold">
            {prefix}{points[hoveredIndex].val}{suffix}
          </span>
        </div>
      )}
    </div>
  );
}
