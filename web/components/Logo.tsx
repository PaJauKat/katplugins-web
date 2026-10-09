import Image from "next/image";
import Link from "next/link";

export default function Logo({
  href = "/",
  className = "",
  showWordmark = true,
}: {
  href?: string;
  className?: string;
  showWordmark?: boolean;
}) {
  return (
    <Link href={href} className={`flex items-center gap-2.5 ${className}`}>
      <Image
        src="/logo-k.png"
        alt="KatPlugins"
        width={36}
        height={29}
        priority
        className="h-8 w-auto"
      />
      {showWordmark && (
        <span className="text-lg font-bold tracking-tight text-white">
          Kat<span className="text-brand">Plugins</span>
        </span>
      )}
    </Link>
  );
}
