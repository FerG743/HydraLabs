import React from 'react';
import { Lock, Eye, EyeOff } from 'lucide-react';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { AUTH_TYPES } from '@/utils/apiConstants';
import { usePasswordVisibility } from '../hooks/useApiRequest';

const QuickAuthSelector = ({ auth, onUpdate, className = '' }) => {
  const { togglePassword, isVisible } = usePasswordVisibility();

  const handleAuthValueChange = (value) => {
    switch (auth.type) {
      case 'bearer':
        onUpdate('auth.bearer', value);
        break;
      case 'basic':
        const [username, password] = value.split(':');
        onUpdate('auth.basic.username', username || '');
        onUpdate('auth.basic.password', password || '');
        break;
      case 'api-key':
        onUpdate('auth.apiKey.value', value);
        break;
    }
  };

  const getAuthValue = () => {
    switch (auth.type) {
      case 'bearer':
        return auth.bearer;
      case 'basic':
        return `${auth.basic.username}:${auth.basic.password}`;
      case 'api-key':
        return auth.apiKey.value;
      default:
        return '';
    }
  };

  const getPlaceholder = () => {
    switch (auth.type) {
      case 'bearer':
        return 'Enter bearer token';
      case 'basic':
        return 'username:password';
      case 'api-key':
        return 'Enter API key';
      default:
        return '';
    }
  };

  const showPasswordToggle = auth.type !== 'none';

  return (
    <div className={`flex items-center gap-2 ${className}`}>
      <Lock className="w-4 h-4 text-muted-foreground flex-shrink-0" />
      
      <Select value={auth.type} onValueChange={(value) => onUpdate('auth.type', value)}>
        <SelectTrigger className="w-[120px] border-0 bg-transparent focus:ring-1 focus:ring-ring">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {AUTH_TYPES.map(type => (
            <SelectItem key={type.value} value={type.value}>
              {type.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      
      {auth.type !== 'none' && (
        <div className="relative flex items-center">
          <Input
            type={isVisible('quick-auth') ? 'text' : 'password'}
            placeholder={getPlaceholder()}
            value={getAuthValue()}
            onChange={(e) => handleAuthValueChange(e.target.value)}
            className="pr-10 min-w-[120px]"
          />
          
          {showPasswordToggle && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => togglePassword('quick-auth')}
              className="absolute right-1 h-7 w-7 p-0"
            >
              {isVisible('quick-auth') ? (
                <EyeOff className="w-3 h-3" />
              ) : (
                <Eye className="w-3 h-3" />
              )}
            </Button>
          )}
        </div>
      )}
    </div>
  );
};

export default QuickAuthSelector;