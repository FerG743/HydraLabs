/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: ["class"],
  content: [
    './index.html',
    './pages/**/*.{js,ts,jsx,tsx}',
    './components/**/*.{ts,tsx}',
    './app/**/*.{ts,tsx}',
    './src/**/*.{js,ts,jsx,tsx}',
  ],
  theme: {
    container: {
      center: true,
      padding: "2rem",
      screens: {
        "2xl": "1400px",
      },
    },
    extend: {
      colors: {
        // Radioactive Sunset custom colors
        'toxic-yellow': '#CCFF00',
        'electric-orange': '#FF6600',
        'hot-red': '#FF0040',
        'amber': '#FFBF00',
        'tangerine': '#FF9500',
        'lemon-cream': '#FFFACD',
        
        // shadcn/ui semantic colors - Radioactive Sunset palette
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        secondary: {
          DEFAULT: "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))",
        },
        destructive: {
          DEFAULT: "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))",
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        accent: {
          DEFAULT: "hsl(var(--accent))",
          foreground: "hsl(var(--accent-foreground))",
        },
        popover: {
          DEFAULT: "hsl(var(--popover))",
          foreground: "hsl(var(--popover-foreground))",
        },
        card: {
          DEFAULT: "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))",
        },
        success: {
          DEFAULT: "hsl(var(--success))",
          foreground: "hsl(var(--success-foreground))",
        },
        warning: {
          DEFAULT: "hsl(var(--warning))",
          foreground: "hsl(var(--warning-foreground))",
        },
        error: {
          DEFAULT: "hsl(var(--error))",
          foreground: "hsl(var(--error-foreground))",
        },
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
      boxShadow: {
        'glow-toxic-yellow': '0 0 20px rgba(204, 255, 0, 0.4)',
        'glow-electric-orange': '0 0 20px rgba(255, 102, 0, 0.4)',
        'glow-hot-red': '0 0 20px rgba(255, 0, 64, 0.4)',
        'glow-amber': '0 0 20px rgba(255, 191, 0, 0.4)',
        'glow-tangerine': '0 0 20px rgba(255, 149, 0, 0.4)',
      },
      keyframes: {
        // shadcn/ui default keyframes
        "accordion-down": {
          from: { height: "0" },
          to: { height: "var(--radix-accordion-content-height)" },
        },
        "accordion-up": {
          from: { height: "var(--radix-accordion-content-height)" },
          to: { height: "0" },
        },
        
        // Custom animation keyframes
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        fadeOut: {
          '0%': { opacity: '1' },
          '100%': { opacity: '0' },
        },
        fadeInUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        fadeInDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        slideInLeft: {
          '0%': { transform: 'translateX(-100%)' },
          '100%': { transform: 'translateX(0)' },
        },
        slideInRight: {
          '0%': { transform: 'translateX(100%)' },
          '100%': { transform: 'translateX(0)' },
        },
        slideUp: {
          '0%': { transform: 'translateY(100%)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        slideDown: {
          '0%': { transform: 'translateY(-100%)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        scaleIn: {
          '0%': { transform: 'scale(0.95)', opacity: '0' },
          '100%': { transform: 'scale(1)', opacity: '1' },
        },
        scaleOut: {
          '0%': { transform: 'scale(1)', opacity: '1' },
          '100%': { transform: 'scale(0.95)', opacity: '0' },
        },
        pulseSubtle: {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.8' },
        },
        bounceGentle: {
          '0%, 20%, 50%, 80%, 100%': { transform: 'translateY(0)' },
          '40%': { transform: 'translateY(-4px)' },
          '60%': { transform: 'translateY(-2px)' },
        },
        shake: {
          '0%, 100%': { transform: 'translateX(0)' },
          '10%, 30%, 50%, 70%, 90%': { transform: 'translateX(-2px)' },
          '20%, 40%, 60%, 80%': { transform: 'translateX(2px)' },
        },
        panelExpand: {
          '0%': { height: '0', opacity: '0' },
          '100%': { height: 'auto', opacity: '1' },
        },
        panelCollapse: {
          '0%': { height: 'auto', opacity: '1' },
          '100%': { height: '0', opacity: '0' },
        },
        testSuccess: {
          '0%': { transform: 'scale(1)', boxShadow: '0 0 0 0 rgba(204, 255, 0, 0.7)' },
          '50%': { transform: 'scale(1.05)', boxShadow: '0 0 20px 10px rgba(204, 255, 0, 0)' },
          '100%': { transform: 'scale(1)', boxShadow: '0 0 0 0 rgba(204, 255, 0, 0)' },
        },
        testError: {
          '0%': { transform: 'scale(1)', boxShadow: '0 0 0 0 rgba(255, 0, 64, 0.7)' },
          '50%': { transform: 'scale(1.05)', boxShadow: '0 0 20px 10px rgba(255, 0, 64, 0)' },
          '100%': { transform: 'scale(1)', boxShadow: '0 0 0 0 rgba(255, 0, 64, 0)' },
        },
        ripple: {
          '0%': { transform: 'scale(0)', opacity: '1' },
          '100%': { transform: 'scale(4)', opacity: '0' },
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' },
        },
        float: {
          '0%, 100%': { transform: 'translateY(0px)' },
          '50%': { transform: 'translateY(-4px)' },
        },
        wiggle: {
          '0%, 100%': { transform: 'rotate(-3deg)' },
          '50%': { transform: 'rotate(3deg)' },
        },
        // Radioactive pulse animations
        pulseToxic: {
          '0%, 100%': { opacity: '1', boxShadow: '0 0 0 0 rgba(204, 255, 0, 0.7)' },
          '50%': { opacity: '.9', boxShadow: '0 0 0 15px rgba(204, 255, 0, 0)' },
        },
        pulseOrange: {
          '0%, 100%': { opacity: '1', boxShadow: '0 0 0 0 rgba(255, 102, 0, 0.7)' },
          '50%': { opacity: '.9', boxShadow: '0 0 0 15px rgba(255, 102, 0, 0)' },
        },
        pulseRed: {
          '0%, 100%': { opacity: '1', boxShadow: '0 0 0 0 rgba(255, 0, 64, 0.7)' },
          '50%': { opacity: '.9', boxShadow: '0 0 0 15px rgba(255, 0, 64, 0)' },
        },
      },
      animation: {
        // shadcn/ui default animations
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
        
        // Fade animations
        'fade-in': 'fadeIn 0.2s ease-in-out',
        'fade-out': 'fadeOut 0.2s ease-in-out',
        'fade-in-up': 'fadeInUp 0.3s ease-out',
        'fade-in-down': 'fadeInDown 0.3s ease-out',
        
        // Slide animations
        'slide-in-left': 'slideInLeft 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'slide-up': 'slideUp 0.2s ease-out',
        'slide-down': 'slideDown 0.2s ease-out',
        
        // Scale animations
        'scale-in': 'scaleIn 0.2s ease-out',
        'scale-out': 'scaleOut 0.2s ease-in',
        'pulse-subtle': 'pulseSubtle 2s infinite',
        
        // Bounce animations
        'bounce-gentle': 'bounceGentle 0.6s ease-out',
        'shake': 'shake 0.5s ease-in-out',
        
        // Loading animations
        'spin-slow': 'spin 3s linear infinite',
        'ping-slow': 'ping 3s cubic-bezier(0, 0, 0.2, 1) infinite',
        
        // App-specific animations
        'panel-expand': 'panelExpand 0.3s ease-out',
        'panel-collapse': 'panelCollapse 0.3s ease-in',
        'test-success': 'testSuccess 0.4s ease-out',
        'test-error': 'testError 0.4s ease-out',
        
        // Extra animations
        'ripple': 'ripple 0.6s linear',
        'shimmer': 'shimmer 2s linear infinite',
        'float': 'float 3s ease-in-out infinite',
        'wiggle': 'wiggle 1s ease-in-out infinite',
        
        // Radioactive animations
        'pulse-toxic': 'pulseToxic 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'pulse-orange': 'pulseOrange 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'pulse-red': 'pulseRed 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
      transitionDuration: {
        '400': '400ms',
        '600': '600ms',
        '800': '800ms',
        '900': '900ms',
      },
      transitionTimingFunction: {
        'bounce-in': 'cubic-bezier(0.68, -0.55, 0.265, 1.55)',
        'smooth': 'cubic-bezier(0.4, 0, 0.2, 1)',
        'snappy': 'cubic-bezier(0.4, 0, 0.1, 1)',
      },
      backgroundImage: {
        'shimmer': 'linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.2), transparent)',
        'gradient-nuclear': 'linear-gradient(135deg, #CCFF00 0%, #FF6600 100%)',
        'gradient-inferno': 'linear-gradient(135deg, #FF6600 0%, #FF0040 100%)',
        'gradient-sunset': 'linear-gradient(135deg, #FFBF00 0%, #FF6600 100%)',
      },
    },
  },
  plugins: [require("tailwindcss-animate")],
}