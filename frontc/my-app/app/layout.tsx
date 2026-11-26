import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "SpeedJato | Agende sua lavagem",
  description: "Cadastre seu veículo e escolha o melhor horário para sua lavagem.",
  icons: { icon: "/lava-icon.svg" },
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="pt-BR">
      <body>{children}</body>
    </html>
  );
}
