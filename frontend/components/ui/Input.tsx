import { cn } from "@/lib/utils";

type InputProps = React.InputHTMLAttributes<HTMLInputElement>;
type TextAreaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;

export function Input({ className, ...props }: InputProps) {
  return <input className={cn("input-base", className)} {...props} />;
}

export function TextArea({ className, ...props }: TextAreaProps) {
  return <textarea className={cn("input-base min-h-30 resize-y", className)} {...props} />;
}
