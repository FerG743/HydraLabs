import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default:
          "bg-primary text-primary-foreground glow-toxic-yellow hover:bg-primary/90 hover:glow-toxic-yellow hover:scale-105 active:scale-95",
        destructive:
          "bg-destructive text-destructive-foreground shadow-glow-hot-red hover:bg-destructive/90 hover:shadow-glow-hot-red hover:scale-105 active:scale-95",
        outline:
          "border border-border bg-background hover:bg-muted hover:text-foreground hover:border-primary transition-all",
        secondary:
          "bg-secondary text-secondary-foreground shadow-glow-amber hover:bg-secondary/80 hover:shadow-glow-amber hover:scale-105 active:scale-95",
        ghost: "hover:bg-muted hover:text-foreground transition-all",
        link: "text-primary underline-offset-4 hover:underline hover:text-primary/80",
        warning:
          "bg-warning text-warning-foreground shadow-glow-electric-orange hover:bg-warning/90 hover:shadow-glow-electric-orange hover:scale-105 active:scale-95",
        success:
          "bg-success text-success-foreground glow-toxic-yellow hover:bg-success/90 hover:glow-toxic-yellow hover:scale-105 active:scale-95",
      },
      size: {
        default: "h-9 px-4 py-2",
        sm: "h-8 rounded-md px-3 text-xs",
        lg: "h-10 rounded-md px-8",
        icon: "h-9 w-9",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button"
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    )
  }
)
Button.displayName = "Button"

export { Button, buttonVariants }