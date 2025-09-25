import React from 'react';
import { FileText, X } from 'lucide-react';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { COMMON_HEADERS } from '@/utils/apiConstants';

const QuickHeaders = ({ headers, onAdd, onRemove, onUpdate, className = '' }) => {
  const enabledHeaders = headers.filter(h => h.key && h.enabled);

  const handleAddHeader = (headerName) => {
    // Check if header already exists
    const existingIndex = headers.findIndex(h => h.key === headerName);
    if (existingIndex !== -1) {
      return; // Header already exists, don't add duplicate
    }

    // Add new header with proper callback structure
    const newHeader = {
      key: headerName,
      value: headerName === 'Content-Type' ? 'application/json' : '',
      enabled: true
    };

    // Call onAdd with the complete header object
    if (onAdd) {
      onAdd(newHeader);
    }
  };

  const handleRemoveHeader = (headerToRemove) => {
    const index = headers.findIndex(h => h === headerToRemove);
    if (index !== -1 && onRemove) {
      onRemove(index);
    }
  };

  const handleEditHeader = (header, field, value) => {
    const index = headers.findIndex(h => h === header);
    if (index !== -1 && onUpdate) {
      onUpdate(index, field, value);
    }
  };

  return (
    <div className={`flex flex-wrap items-center gap-2 ${className}`}>
      <FileText className="w-4 h-4 text-muted-foreground flex-shrink-0" />
      
      <Select onValueChange={handleAddHeader}>
        <SelectTrigger className="w-[140px] border-0 bg-transparent focus:ring-1 focus:ring-ring">
          <SelectValue placeholder="Add Header" />
        </SelectTrigger>
        <SelectContent>
          {COMMON_HEADERS.filter(header => 
            // Don't show headers that are already added
            !enabledHeaders.some(h => h.key === header)
          ).map(header => (
            <SelectItem key={header} value={header}>
              {header}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      
      <div className="flex flex-wrap gap-1">
        {enabledHeaders.map((header, i) => (
          <Badge 
            key={`${header.key}-${i}`}
            variant="secondary"
            className="animate-scale-in group cursor-pointer"
          >
            <span className="font-medium">{header.key}</span>
            {header.value && (
              <span className="text-muted-foreground ml-1">
                : {header.value.length > 20 
                    ? `${header.value.substring(0, 20)}...` 
                    : header.value
                  }
              </span>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={() => handleRemoveHeader(header)}
              className="h-4 w-4 p-0 ml-1 hover:bg-destructive hover:text-destructive-foreground opacity-0 group-hover:opacity-100 transition-opacity"
              title="Remove header"
            >
              <X className="w-3 h-3" />
            </Button>
          </Badge>
        ))}
      </div>
      
      {enabledHeaders.length === 0 && (
        <span className="text-xs text-muted-foreground italic">No headers added</span>
      )}
    </div>
  );
};

export default QuickHeaders;