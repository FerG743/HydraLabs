import React, { useRef, useEffect } from 'react';

const ScrollableContainer = ({ 
  children, 
  className = '',
  scrollbarWidth = 8,
  ...props 
}) => {
  const containerRef = useRef(null);
  const scopeId = useRef(`scrollbar-${Math.random().toString(36).substr(2, 9)}`);

  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.setAttribute('data-scrollbar', scopeId.current);
    }
  }, []);

  return (
    <>
      <style>{`
        [data-scrollbar="${scopeId.current}"]::-webkit-scrollbar {
          width: ${scrollbarWidth}px;
        }
        [data-scrollbar="${scopeId.current}"]::-webkit-scrollbar-track {
          background: transparent;
        }
        [data-scrollbar="${scopeId.current}"]::-webkit-scrollbar-track:active {
          background: transparent;
        }
        [data-scrollbar="${scopeId.current}"]::-webkit-scrollbar-thumb {
          background: hsl(var(--muted-foreground) / 0.2);
          border-radius: 4px;
          border: 2px solid transparent;
          background-clip: padding-box;
          transition: background 0.2s ease, box-shadow 0.2s ease;
        }
        [data-scrollbar="${scopeId.current}"]::-webkit-scrollbar-thumb:hover {
          background: hsl(var(--muted-foreground) / 0.35);
          background-clip: padding-box;
          box-shadow: 0 0 8px hsl(var(--primary) / 0.2);
        }
        [data-scrollbar="${scopeId.current}"]::-webkit-scrollbar-thumb:active {
          background: hsl(var(--muted-foreground) / 0.5);
          background-clip: padding-box;
          box-shadow: 0 0 12px hsl(var(--primary) / 0.4);
        }
        [data-scrollbar="${scopeId.current}"]::-webkit-scrollbar-button {
          display: none;
        }
      `}</style>
      <div 
        ref={containerRef}
        className={`overflow-y-auto outline-none ${className}`}
        {...props}
      >
        {children}
      </div>
    </>
  );
};

export default ScrollableContainer;