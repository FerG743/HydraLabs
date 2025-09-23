import React from 'react';
import { Plus, X, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';

const KeyValueEditor = ({
  items = [],
  onAdd,
  onRemove,
  onUpdate,
  onClear,
  placeholder = { key: 'Key', value: 'Value' },
  showEnabled = true,
  showType = false,
  className = '',
  emptyMessage = 'No items added yet',
  addButtonText = 'Add Item'
}) => {
  const hasItems = items.length > 0;
  const hasEnabledItems = items.some(item => item.enabled && item.key);

  return (
    <div className={`space-y-2 ${className}`}>
      {hasItems ? (
        <>
          {items.map((item, index) => (
            <div 
              key={index} 
              className={`flex items-center gap-2 animate-fade-in-up transition-all duration-200 ${
                !item.enabled ? 'opacity-50' : ''
              }`}
            >
              {showEnabled && (
                <Checkbox
                  checked={item.enabled}
                  onCheckedChange={(checked) => onUpdate(index, 'enabled', checked)}
                />
              )}
              
              <Input
                placeholder={placeholder.key}
                value={item.key}
                onChange={(e) => onUpdate(index, 'key', e.target.value)}
                className="flex-1"
              />
              
              <Input
                placeholder={placeholder.value}
                value={item.value}
                onChange={(e) => onUpdate(index, 'value', e.target.value)}
                className="flex-1"
              />
              
              {showType && (
                <Select
                  value={item.type || 'text'}
                  onValueChange={(value) => onUpdate(index, 'type', value)}
                >
                  <SelectTrigger className="w-20">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="text">Text</SelectItem>
                    <SelectItem value="file">File</SelectItem>
                  </SelectContent>
                </Select>
              )}
              
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onRemove(index)}
                className="text-destructive hover:text-destructive hover:bg-destructive/10"
              >
                <X className="w-4 h-4" />
              </Button>
            </div>
          ))}
          
          <div className="flex items-center justify-between pt-2">
            <Button
              variant="outline"
              size="sm"
              onClick={onAdd}
            >
              <Plus className="w-4 h-4 mr-2" />
              {addButtonText}
            </Button>
            
            {hasEnabledItems && (
              <Button
                variant="ghost" 
                size="sm"
                onClick={onClear}
                className="text-muted-foreground hover:text-destructive"
              >
                <Trash2 className="w-3 h-3 mr-1" />
                Clear
              </Button>
            )}
          </div>
        </>
      ) : (
        <div className="text-center py-8 text-muted-foreground">
          <div className="text-sm mb-3">{emptyMessage}</div>
          <Button variant="outline" onClick={onAdd}>
            <Plus className="w-4 h-4 mr-2" />
            {addButtonText}
          </Button>
        </div>
      )}
    </div>
  );
};

export default KeyValueEditor;