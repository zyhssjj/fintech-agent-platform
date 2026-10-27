import React, { useState, useEffect } from "react";

interface AnimatedNumberProps {
  value: number;
  decimals?: number;
  suffix?: string;
  prefix?: string;
  duration?: number;
}

export default function AnimatedNumber({ 
  value, 
  decimals = 1, 
  suffix = "", 
  prefix = "", 
  duration = 1000 
}: AnimatedNumberProps) {
  const [current, setCurrent] = useState(0);

  useEffect(() => {
    let startTimestamp: number | null = null;
    const step = (timestamp: number) => {
      if (!startTimestamp) startTimestamp = timestamp;
      const progress = Math.min((timestamp - startTimestamp) / duration, 1);
      
      // Ease-out cubic for a ultra-smooth high-end deceleration effect
      const easeProgress = 1 - Math.pow(1 - progress, 3);
      
      setCurrent(easeProgress * value);
      if (progress < 1) {
        window.requestAnimationFrame(step);
      }
    };
    window.requestAnimationFrame(step);
  }, [value, duration]);

  return (
    <span className="font-mono">
      {prefix}
      {current.toLocaleString(undefined, {
        minimumFractionDigits: decimals,
        maximumFractionDigits: decimals
      })}
      {suffix}
    </span>
  );
}
