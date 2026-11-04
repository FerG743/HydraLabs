import React from 'react';
import { Play } from 'lucide-react';
import { Button } from '@/components/ui/button';

const SendButton = ({ 
  onSend, 
  isLoading, 
  disabled, 
  className = '',
  loadingText = 'Sending...',
  children = 'Send',
  variant = 'default',
  size = 'default'
}) => {
  const isDisabled = isLoading || disabled;

  return (
    <Button
      onClick={onSend}
      disabled={isDisabled}
      variant={variant}
      size={size}
      className={className}
    >
      {isLoading ? (
        <>
          <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin mr-2" />
          {loadingText}
        </>
      ) : (
        <>
          <Play className="w-4 h-4 mr-2" />
          {children}
        </>
      )}
    </Button>
  );
};

export default SendButton;