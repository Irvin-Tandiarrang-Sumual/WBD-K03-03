import Link from "next/link";
import type { ComponentProps, ReactNode } from "react";

interface ButtonLinkProps extends Omit<ComponentProps<typeof Link>, "className"> {
  children: ReactNode;
  variant?: "primary" | "ghost" | "secondary";
  size?: "sm" | "md" | "lg";
  className?: string;
}

export default function ButtonLink({
  href,
  variant = "primary",
  size = "md",
  children,
  className = "",
  ...rest
}: ButtonLinkProps) {
  return (
    <Link href={href} className={`btn btn-${variant} btn-${size} ${className}`.trim()} {...rest}>
      {children}
    </Link>
  );
}
