import React from 'react';
import { Settings, X, Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

const QuickParams = ({ params, onAdd, onRemove, className = '' }) => {
  const enabledParams = params.filter(p => p.key && p.enabled);

  const handleRemoveParam = (paramToRemove) => {
    const index = params.findIndex(p => p === paramToRemove);
    if (index !== -1) {
      onRemove(index);
    }
  };

  return (
    <div className={`flex flex-wrap items-center gap-2 ${className}`}>
      <Settings className="w-4 h-4 text-muted-foreground flex-shrink-0" />
      
      <Button
        variant="ghost"
        size="sm"
        onClick={onAdd}
        className="text-sm h-8 px-2"
      >
        <Plus className="w-3 h-3 mr-1" />
        Add Param
      </Button>
      
      <div className="flex flex-wrap gap-1">
        {enabledParams.map((param, i) => (
          <Badge 
            key={`${param.key}-${i}`}
            variant="outline"
            className="animate-scale-in"
          >
            <span className="font-medium">{param.key}</span>
            <span className="text-muted-foreground">=</span>
            <span>
              {param.value.length > 15 
                ? `${param.value.substring(0, 15)}...` 
                : param.value || '""'
              }
            </span>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => handleRemoveParam(param)}
              className="h-4 w-4 p-0 ml-1 hover:bg-transparent"
            >
              <X className="w-3 h-3" />
            </Button>
          </Badge>
        ))}
      </div>
      
      {enabledParams.length === 0 && (
        <span className="text-xs text-muted-foreground italic">No parameters added</span>
      )}
    </div>
  );
};

export default QuickParams;