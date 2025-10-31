import React, { useRef, useEffect, useState } from 'react';

const AnimatedCollapse = ({ 
  isOpen, 
  children, 
  duration = 300,
  staggerDelay = 30,
  staggerChildren = false 
}) => {
  const contentRef = useRef(null);
  const innerRef = useRef(null);
  const [height, setHeight] = useState(0);
  const [isAnimating, setIsAnimating] = useState(false);
  
  useEffect(() => {
    const measureHeight = () => {
      if (innerRef.current) {
        const measuredHeight = innerRef.current.offsetHeight;
        setHeight(isOpen ? measuredHeight : 0);
      }
    };

    setIsAnimating(true);
    measureHeight();
    
    const timer = setTimeout(measureHeight, 10);
    
    // Mark animation as complete after duration
    const animationTimer = setTimeout(() => {
      setIsAnimating(false);
    }, duration);
    
    return () => {
      clearTimeout(timer);
      clearTimeout(animationTimer);
    };
  }, [isOpen, children, duration]);
  
  return (
    <div 
      ref={contentRef}
      style={{ 
        height: `${height}px`,
        maxHeight: isOpen && !isAnimating ? 'none' : undefined,
        overflow: isAnimating ? 'hidden' : (isOpen ? 'visible' : 'hidden'),
        transition: `height ${duration}ms cubic-bezier(0.4, 0, 0.2, 1)`,
      }}
    >
      <div ref={innerRef}>
        {staggerChildren ? (
          React.Children.map(children, (child, idx) => (
            <div
              key={idx}
              style={{ 
                opacity: isOpen ? 1 : 0,
                transform: isOpen ? 'translateY(0)' : 'translateY(-4px)',
                transition: `opacity 0.2s ease-out ${idx * staggerDelay}ms, transform 0.2s ease-out ${idx * staggerDelay}ms`,
              }}
            >
              {child}
            </div>
          ))
        ) : children}
      </div>
    </div>
  );
};

export { AnimatedCollapse };
export default AnimatedCollapse;