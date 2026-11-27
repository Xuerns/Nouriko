"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  ChevronDown,
  Globe,
  LogOut,
  Moon,
  Sun,
  User,
} from "lucide-react";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { useTheme } from "@/components/providers/ThemeProvider";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";

export function AccountMenu() {
  const pathname = usePathname();
  const { isAuthenticated, isLoading, profile, logout } = useAuth();
  const { language, t, toggleLanguage } = useI18n();
  const { theme, toggleTheme } = useTheme();

  const isProfileActive = pathname === "/profile";
  const displayName = profile?.display_name?.trim() ?? "";
  const isSafeAvatarUrl =
    typeof profile?.profile_url === "string" &&
    profile.profile_url.startsWith("https://api.dicebear.com/");
  const initials = (displayName || profile?.email || "N")
    .slice(0, 1)
    .toUpperCase();

  if (isLoading) {
    return (
      <div
        aria-hidden="true"
        className="flex min-h-10 items-center gap-1.5 px-1 py-1 sm:pr-2"
      >
        <Skeleton className="size-8 rounded-full" />
        <Skeleton className="hidden h-3.5 w-16 rounded-md sm:block" />
      </div>
    );
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            aria-label={displayName || t.nav.profile}
            className="min-h-10 h-auto rounded-full border-0 bg-transparent px-1 py-1 gap-1.5 text-foreground shadow-none transition-colors duration-150 hover:bg-muted/60 hover:text-foreground aria-expanded:bg-muted/60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--brand-primary)] focus-visible:ring-0 sm:pr-2 dark:hover:bg-muted/40 dark:aria-expanded:bg-muted/40"
          />
        }
      >
        <Avatar
          className={`size-8 transition-opacity duration-150 group-hover/button:opacity-90 ${
            isProfileActive
              ? "outline-2 outline-offset-2 outline-[var(--brand-primary)]"
              : ""
          }`}
        >
          {isSafeAvatarUrl && profile?.profile_url ? (
            <AvatarImage src={profile.profile_url} alt="" />
          ) : null}
          <AvatarFallback className="text-xs font-semibold">
            {initials}
          </AvatarFallback>
        </Avatar>
        {displayName ? (
          <span
            className={`hidden sm:inline max-w-28 truncate text-xs ${
              isProfileActive
                ? "font-semibold text-[var(--brand-primary)]"
                : "font-medium text-foreground"
            }`}
          >
            {displayName}
          </span>
        ) : null}
        <ChevronDown
          className="size-3.5 shrink-0 stroke-[1.75] text-muted-foreground transition-transform duration-150 group-aria-expanded/button:rotate-180"
          aria-hidden="true"
        />
      </DropdownMenuTrigger>

      <DropdownMenuContent
        align="end"
        sideOffset={8}
        className="w-56 rounded-2xl border border-border bg-card p-1.5 text-card-foreground shadow-none ring-0"
      >
        {profile ? (
          <DropdownMenuGroup>
            <DropdownMenuLabel className="px-2.5 py-2">
              {displayName ? (
                <p className="truncate text-xs font-semibold text-foreground">
                  {displayName}
                </p>
              ) : null}
              <p className="truncate text-[11px] font-normal text-muted-foreground">
                {profile.email}
              </p>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
          </DropdownMenuGroup>
        ) : null}

        <DropdownMenuItem
          render={<Link href="/profile" />}
          className={`rounded-xl px-2.5 py-2 text-xs ${
            isProfileActive
              ? "bg-[var(--brand-tint)] font-semibold text-[var(--brand-primary)]"
              : "font-medium"
          }`}
        >
          <User className="size-4 stroke-[1.75]" aria-hidden="true" />
          <span>{t.nav.profile}</span>
        </DropdownMenuItem>

        <DropdownMenuSeparator />

        <DropdownMenuItem
          onClick={toggleLanguage}
          className="rounded-xl px-2.5 py-2 text-xs font-medium"
        >
          <Globe className="size-4 stroke-[1.75]" aria-hidden="true" />
          <span>{t.common.languageLabel}</span>
          <span className="ml-auto text-[11px] font-semibold tabular-nums text-muted-foreground">
            {language === "id" ? "ID" : "EN"}
          </span>
        </DropdownMenuItem>

        <DropdownMenuItem
          onClick={(event) => toggleTheme(event)}
          className="rounded-xl px-2.5 py-2 text-xs font-medium"
        >
          {theme === "dark" ? (
            <Moon className="size-4 stroke-[1.75]" aria-hidden="true" />
          ) : (
            <Sun className="size-4 stroke-[1.75]" aria-hidden="true" />
          )}
          <span>{t.common.themeLabel}</span>
          <span className="ml-auto text-[11px] text-muted-foreground">
            {theme === "dark" ? t.common.themeDark : t.common.themeLight}
          </span>
        </DropdownMenuItem>

        {!isLoading && isAuthenticated ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              variant="destructive"
              onClick={() => void logout()}
              className="rounded-xl px-2.5 py-2 text-xs font-medium"
            >
              <LogOut className="size-4 stroke-[1.75]" aria-hidden="true" />
              <span>{t.nav.logout}</span>
            </DropdownMenuItem>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
