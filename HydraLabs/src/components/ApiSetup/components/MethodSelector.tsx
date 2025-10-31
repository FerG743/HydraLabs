import React from 'react';
import { Button } from '@/components/ui/button';
import { HTTP_METHODS } from '@/utils/apiConstants';

const MethodSelector = ({ method, onChange, className = '' }) => {
  return (
    <div className={`flex gap-1 ${className}`}>
      {HTTP_METHODS.map(httpMethod => (
        <Button
          key={httpMethod}
          onClick={() => onChange(httpMethod)}
          variant={method === httpMethod ? "default" : "outline"}
          size="sm"
          className="text-xs font-medium"
        >
          {httpMethod}
        </Button>
      ))}
    </div>
  );
};

export default MethodSelector;