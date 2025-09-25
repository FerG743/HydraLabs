import React from 'react';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from 'lucide-react';
import { HTTP_METHODS } from '@/utils/apiConstants';

const MethodSelector = ({ method, onChange, className = '' }) => {
  const getMethodVariant = (method) => {
    switch(method) {
      case 'GET': return 'secondary';
      case 'POST': return 'default';
      case 'PUT': return 'secondary';
      case 'PATCH': return 'outline';
      case 'DELETE': return 'destructive';
      default: return 'outline';
    }
  };

  return (
    <Select value={method} onValueChange={onChange}>
      <SelectTrigger className={`w-[100px] ${className}`}>
        <Badge variant={getMethodVariant(method)} className="text-xs">
          {method}
        </Badge>
      </SelectTrigger>
      <SelectContent>
        {HTTP_METHODS.map(httpMethod => (
          <SelectItem key={httpMethod} value={httpMethod}>
            <Badge variant={getMethodVariant(httpMethod)} className="text-xs">
              {httpMethod}
            </Badge>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};

export default MethodSelector;