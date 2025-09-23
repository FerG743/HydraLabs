import React, { useState, useEffect } from 'react';
import { AlertCircle, Globe } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { validateUrl } from '../utils/apiHelpers';

const UrlInput = ({ 
  value, 
  onChange, 
  placeholder = "https://api.example.com/endpoint",
  className = '',
  showValidation = true
}) => {
  const [validation, setValidation] = useState({ isValid: true, error: null });

  useEffect(() => {
    if (value && showValidation) {
      const result = validateUrl(value);
      setValidation(result);
    } else {
      setValidation({ isValid: true, error: null });
    }
  }, [value, showValidation]);

  return (
    <div className="relative flex-1">
      <div className="relative">
        <Globe className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <Input
          type="text"
          placeholder={placeholder}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className={`pl-10 ${
            !validation.isValid && value ? 'border-destructive focus-visible:ring-destructive' : ''
          } ${className}`}
        />
        {!validation.isValid && value && (
          <AlertCircle className="absolute right-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-destructive" />
        )}
      </div>
      
      {!validation.isValid && validation.error && (
        <Alert variant="destructive" className="mt-2">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{validation.error}</AlertDescription>
        </Alert>
      )}
    </div>
  );
};

export default UrlInput;