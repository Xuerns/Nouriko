"use client";

import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type KeyboardEvent,
  type MouseEvent,
  type PointerEvent as ReactPointerEvent,
} from "react";
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

type ExtendedFocusOptions = FocusOptions & { focusVisible?: boolean };

function focusElement(
  el: HTMLElement | null,
  options?: ExtendedFocusOptions
) {
  if (!el) return;
  try {
    el.focus(options);
  } catch {
    el.focus({ preventScroll: options?.preventScroll });
  }
}

export function AccountMenu() {
  const pathname = usePathname();
  const { isAuthenticated, isLoading, profile, logout } = useAuth();
  const { language, t, toggleLanguage } = useI18n();
  const { theme, toggleTheme } = useTheme();

  const triggerId = useId();
  const menuId = useId();

  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const menuPopoverRef = useRef<HTMLDivElement | null>(null);
  const menuGroupRef = useRef<HTMLDivElement | null>(null);
  const pointerTriggerRef = useRef(false);
  const enterOnOpenRef = useRef(false);

  const [isOpen, setIsOpen] = useState(false);

  const getEnabledItems = useCallback((): HTMLElement[] => {
    const group = menuGroupRef.current;
    if (!group) return [];
    return Array.from(
      group.querySelectorAll<HTMLElement>("a[href], button")
    ).filter((item) => {
      if (item instanceof HTMLButtonElement && item.disabled) return false;
      return item.getAttribute("aria-disabled") !== "true";
    });
  }, []);

  const isPopoverOpen = useCallback((el: HTMLDivElement | null): boolean => {
    if (!el) return false;
    try {
      return el.matches(":popover-open");
    } catch {
      return el.classList.contains(":popover-open");
    }
  }, []);

  const closeMenuAndRestore = useCallback(
    (fromPointer: boolean) => {
      const popover = menuPopoverRef.current;
      const trigger = triggerRef.current;
      if (!popover || !isPopoverOpen(popover)) return;

      focusElement(trigger, {
        preventScroll: true,
        focusVisible: !fromPointer,
      });
      try {
        popover.hidePopover();
      } catch {
        // Ignore if popover is already hidden.
      }
    },
    [isPopoverOpen]
  );

  const positionMenuFallback = useCallback(() => {
    const popover = menuPopoverRef.current;
    const trigger = triggerRef.current;
    if (!popover || !trigger) return;
    if (
      typeof CSS !== "undefined" &&
      typeof CSS.supports === "function" &&
      CSS.supports("position-area", "block-end")
    ) {
      return;
    }

    const triggerRect = trigger.getBoundingClientRect();
    const menuRect = popover.getBoundingClientRect();
    const gap = 8;
    const viewportPadding = 8;

    let top = triggerRect.bottom + gap;
    if (top + menuRect.height > window.innerHeight - viewportPadding) {
      top = Math.max(viewportPadding, triggerRect.top - menuRect.height - gap);
    }

    const left = Math.max(
      viewportPadding,
      Math.min(
        triggerRect.right - menuRect.width,
        window.innerWidth - menuRect.width - viewportPadding
      )
    );

    popover.style.left = `${Math.round(left)}px`;
    popover.style.top = `${Math.round(top)}px`;
  }, []);

  // Fallback roles & arrow-key navigation when native focusgroup is unsupported.
  useEffect(() => {
    const group = menuGroupRef.current;
    if (!group || "focusGroup" in HTMLElement.prototype) return;

    group.setAttribute("role", "menu");
    group.querySelectorAll("a[href]:not([role]), button:not([role])").forEach(
      (item) => {
        item.setAttribute("role", "menuitem");
      }
    );

    const handleGroupKeyDown = (event: globalThis.KeyboardEvent) => {
      if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
      const items = getEnabledItems();
      if (items.length === 0) return;

      const active = document.activeElement as HTMLElement | null;
      const current = active ? items.indexOf(active) : -1;
      if (current < 0) return;

      const next =
        event.key === "Home"
          ? 0
          : event.key === "End"
            ? items.length - 1
            : (current + (event.key === "ArrowUp" ? -1 : 1) + items.length) %
              items.length;

      event.preventDefault();
      event.stopImmediatePropagation();
      items[next]?.focus({ preventScroll: true });
    };

    group.addEventListener("keydown", handleGroupKeyDown, true);
    return () => {
      group.removeEventListener("keydown", handleGroupKeyDown, true);
    };
  }, [getEnabledItems]);

  // Popover toggle lifecycle, fallback positioning, Escape/Tab, and outside focus handling.
  useEffect(() => {
    const popover = menuPopoverRef.current;
    const trigger = triggerRef.current;
    if (!popover || !trigger) return;

    const handleToggle = (event: Event) => {
      const toggleEvent = event as ToggleEvent;
      const open = toggleEvent.newState === "open";
      setIsOpen(open);

      if (open) {
        positionMenuFallback();
        if (enterOnOpenRef.current) {
          enterOnOpenRef.current = false;
          getEnabledItems()[0]?.focus({ preventScroll: true });
        } else if (pointerTriggerRef.current) {
          pointerTriggerRef.current = false;
          focusElement(trigger, { preventScroll: true, focusVisible: false });
        }
      } else {
        enterOnOpenRef.current = false;
        pointerTriggerRef.current = false;
      }
    };

    const handleMenuKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        closeMenuAndRestore(false);
        return;
      }
      if (event.key === "Tab" && isPopoverOpen(popover)) {
        try {
          popover.hidePopover();
        } catch {
          // Ignore if already closed.
        }
      }
    };

    const handleDocumentFocusIn = (event: FocusEvent) => {
      if (!isPopoverOpen(popover)) return;
      const target = event.target as Node | null;
      if (target && !popover.contains(target) && target !== trigger) {
        try {
          popover.hidePopover();
        } catch {
          // Ignore if already closed.
        }
      }
    };

    popover.addEventListener("toggle", handleToggle);
    popover.addEventListener("keydown", handleMenuKeyDown);
    document.addEventListener("focusin", handleDocumentFocusIn);

    return () => {
      popover.removeEventListener("toggle", handleToggle);
      popover.removeEventListener("keydown", handleMenuKeyDown);
      document.removeEventListener("focusin", handleDocumentFocusIn);
    };
  }, [
    closeMenuAndRestore,
    getEnabledItems,
    isPopoverOpen,
    positionMenuFallback,
  ]);

  function handleTriggerPointerDown(e: ReactPointerEvent<HTMLButtonElement>) {
    e.preventDefault();
    pointerTriggerRef.current = true;
  }

  function handleTriggerClick(e: MouseEvent<HTMLButtonElement>) {
    const popover = menuPopoverRef.current;
    const trigger = triggerRef.current;
    if (!popover || !trigger) return;

    const fromKeyboard = e.detail === 0;
    if (fromKeyboard) {
      pointerTriggerRef.current = false;
    }

    if ("commandForElement" in HTMLButtonElement.prototype) {
      if (fromKeyboard && !isPopoverOpen(popover)) {
        enterOnOpenRef.current = true;
      }
      return;
    }

    e.preventDefault();
    if (isPopoverOpen(popover)) {
      closeMenuAndRestore(!fromKeyboard);
      return;
    }

    enterOnOpenRef.current = fromKeyboard;
    try {
      (
        popover as HTMLDivElement & {
          showPopover: (options?: { source?: HTMLElement }) => void;
        }
      ).showPopover({ source: trigger });
    } catch {
      popover.showPopover();
    }
  }

  function handleTriggerKeyDown(e: KeyboardEvent<HTMLButtonElement>) {
    if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
    const popover = menuPopoverRef.current;
    const trigger = triggerRef.current;
    if (!popover || !trigger) return;

    e.preventDefault();
    if (isPopoverOpen(popover)) {
      const items = getEnabledItems();
      const target =
        e.key === "ArrowUp" ? items[items.length - 1] : items[0];
      target?.focus({ preventScroll: true });
      return;
    }

    enterOnOpenRef.current = true;
    pointerTriggerRef.current = false;
    trigger.click();
  }

  function handleProfileLinkClick() {
    const popover = menuPopoverRef.current;
    if (popover && isPopoverOpen(popover)) {
      try {
        popover.hidePopover();
      } catch {
        // Ignore if already closed.
      }
    }
  }

  function handleLanguageClick(e: MouseEvent<HTMLButtonElement>) {
    toggleLanguage();
    closeMenuAndRestore(e.detail !== 0);
  }

  function handleThemeClick(e: MouseEvent<HTMLButtonElement>) {
    toggleTheme();
    closeMenuAndRestore(e.detail !== 0);
  }

  function handleLogoutClick(e: MouseEvent<HTMLButtonElement>) {
    closeMenuAndRestore(e.detail !== 0);
    void logout();
  }

  const isProfileActive = pathname === "/profile";

  return (
    <div className="flex items-center">
      <button
        ref={triggerRef}
        id={triggerId}
        type="button"
        aria-haspopup="menu"
        aria-expanded={isOpen}
        aria-label={t.nav.profile}
        title={t.nav.profile}
        onPointerDown={handleTriggerPointerDown}
        onClick={handleTriggerClick}
        onKeyDown={handleTriggerKeyDown}
        {...({
          commandfor: menuId,
          command: "toggle-popover",
        } as Record<string, unknown>)}
        className={`inline-flex h-9 items-center gap-1.5 rounded-md border px-2 text-xs font-medium transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 aria-expanded:bg-neutral-100 dark:aria-expanded:bg-neutral-900 ${
          isProfileActive
            ? "border-neutral-900 dark:border-neutral-100 text-neutral-900 dark:text-neutral-100"
            : "border-neutral-300 dark:border-neutral-700 text-neutral-800 dark:text-neutral-200"
        }`}
      >
        {profile?.profile_url ? (
          /* eslint-disable-next-line @next/next/no-img-element */
          <img
            src={profile.profile_url}
            alt=""
            aria-hidden="true"
            className="h-5 w-5 shrink-0 rounded-sm border border-neutral-300 dark:border-neutral-700 object-cover"
          />
        ) : (
          <User className="h-4 w-4 shrink-0" aria-hidden="true" />
        )}
        <ChevronDown className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
      </button>

      <div
        ref={menuPopoverRef}
        id={menuId}
        popover="auto"
        className="account-menu-popover min-w-48 rounded-lg border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-950 p-1 text-neutral-900 dark:text-neutral-100"
      >
        <div
          ref={menuGroupRef}
          aria-labelledby={triggerId}
          {...({ focusgroup: "menu nomemory" } as Record<string, unknown>)}
          className="flex flex-col gap-0.5"
        >
          <Link
            href="/profile"
            aria-current={isProfileActive ? "page" : undefined}
            onClick={handleProfileLinkClick}
            className={`flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-left text-xs transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 focus-visible:bg-neutral-100 dark:focus-visible:bg-neutral-900 focus:outline-none ${
              isProfileActive
                ? "font-semibold text-neutral-900 dark:text-neutral-100"
                : "font-medium text-neutral-800 dark:text-neutral-200"
            }`}
          >
            <User className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span>{t.nav.profile}</span>
          </Link>

          <hr className="my-1 border-neutral-200 dark:border-neutral-800" />

          <button
            type="button"
            onClick={handleLanguageClick}
            className="flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-left text-xs font-medium text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 focus-visible:bg-neutral-100 dark:focus-visible:bg-neutral-900 focus:outline-none"
          >
            <Globe className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span>{t.common.languageLabel}</span>
            <span className="ml-auto font-mono text-[11px] text-neutral-500 dark:text-neutral-400">
              {language === "id" ? "ID" : "EN"}
            </span>
          </button>

          <button
            type="button"
            onClick={handleThemeClick}
            className="flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-left text-xs font-medium text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 focus-visible:bg-neutral-100 dark:focus-visible:bg-neutral-900 focus:outline-none"
          >
            {theme === "dark" ? (
              <Moon className="h-4 w-4 shrink-0" aria-hidden="true" />
            ) : (
              <Sun className="h-4 w-4 shrink-0" aria-hidden="true" />
            )}
            <span>{t.common.themeLabel}</span>
            <span className="ml-auto text-[11px] text-neutral-500 dark:text-neutral-400">
              {theme === "dark" ? t.common.themeDark : t.common.themeLight}
            </span>
          </button>

          {!isLoading && isAuthenticated ? (
            <>
              <hr className="my-1 border-neutral-200 dark:border-neutral-800" />
              <button
                type="button"
                onClick={handleLogoutClick}
                className="flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-left text-xs font-medium text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 focus-visible:bg-neutral-100 dark:focus-visible:bg-neutral-900 focus:outline-none"
              >
                <LogOut className="h-4 w-4 shrink-0" aria-hidden="true" />
                <span>{t.nav.logout}</span>
              </button>
            </>
          ) : null}
        </div>
      </div>
    </div>
  );
}
