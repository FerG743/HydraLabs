import React, { useRef, useEffect, useState } from 'react';

const AnimatedCollapse = ({ 
  isOpen, 
  children, 
  duration = 300,
  staggerDelay = 30,
  staggerChildren = false 
}) => {
  const contentRef = useRef(null);
  const [height, setHeight] = useState(0);
  
  useEffect(() => {
    if (contentRef.current) {
      setHeight(isOpen ? contentRef.current.scrollHeight : 0);
    }
  }, [isOpen, children]);
  
  return (
    <div 
      ref={contentRef}
      style={{ 
        height: `${height}px`,
        overflow: 'hidden',
        transition: `height ${duration}ms cubic-bezier(0.4, 0, 0.2, 1)`,
      }}
    >
      <div>
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