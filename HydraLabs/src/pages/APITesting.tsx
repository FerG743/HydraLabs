import React from 'react';
import ActivityPanel from '@/components/ActivityPanel/ActivityPanel';

const APITesting = () => {
  return (
    // Apply the background directly using your CSS variables
    <div 
      className="min-h-screen p-6 space-y-6" 
      style={{ 
        backgroundColor: 'hsl(300, 15%, 6%)', // Your --background value
        color: 'hsl(60, 9%, 98%)'  // Your --foreground value
      }}
    >
      <h1 style={{ color: 'hsl(75, 100%, 50%)' }}>Testing</h1> {/* Your --primary */}
      <div style={{ 
        backgroundColor: 'hsl(315, 20%, 9%)', // Your --card value
        color: 'hsl(60, 9%, 98%)',
        borderRadius: '0.75rem',
        padding: '1rem'
      }}>
        <ActivityPanel />
      </div>
    </div>
  );
};

export default APITesting;