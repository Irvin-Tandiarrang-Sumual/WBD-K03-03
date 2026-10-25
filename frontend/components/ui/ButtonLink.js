import Link from "next/link";

export default function ButtonLink({
  href,
  variant = "primary",
  size = "md",
  children,
  className = "",
  ...rest
}) {
  return (
    <Link href={href} className={`btn btn-${variant} btn-${size} ${className}`.trim()} {...rest}>
      {children}
    </Link>
  );
}
