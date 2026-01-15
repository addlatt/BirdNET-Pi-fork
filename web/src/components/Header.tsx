import { useState, useEffect, useRef } from 'preact/hooks';
import type { JSX } from 'preact';

/**
 * Navigation link type
 */
interface NavLink {
  href: string;
  label: string;
}

/**
 * Header component with navigation.
 */
export function Header(): JSX.Element {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const settingsRef = useRef<HTMLDivElement>(null);

  const navLinks: NavLink[] = [
    { href: '/app/', label: 'Overview' },
    { href: '/app/live', label: 'Live' },
    { href: '/app/detections', label: 'Detections' },
    { href: '/app/recordings', label: 'Play' },
    { href: '/app/stats', label: 'Stats' },
    { href: '/app/species', label: 'Species' },
  ];

  const settingsLinks: NavLink[] = [
    { href: '/app/settings', label: 'Settings' },
    { href: '/app/advanced-settings', label: 'Advanced' },
    { href: '/app/services', label: 'Services' },
  ];

  // Close settings dropdown when clicking/touching outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent | TouchEvent) {
      if (settingsRef.current && !settingsRef.current.contains(event.target as Node)) {
        setIsSettingsOpen(false);
      }
    }
    // Add both mousedown and touchstart for cross-device support
    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('touchstart', handleClickOutside, { passive: true });
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('touchstart', handleClickOutside);
    };
  }, []);

  return (
    <header class="bg-primary-600 text-white shadow-lg">
      <div class="container mx-auto px-4">
        <div class="flex items-center justify-between h-16">
          {/* Logo */}
          <a href="/app/" class="flex items-center space-x-2">
            <svg class="w-8 h-8" fill="currentColor" viewBox="0 0 24 24">
              <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
            </svg>
            <span class="font-bold text-xl">BirdNET-Pi</span>
          </a>

          {/* Desktop Navigation */}
          <nav class="hidden md:flex items-center space-x-8">
            {navLinks.map((link) => (
              <a
                key={link.href}
                href={link.href}
                class="hover:text-primary-200 transition-colors font-medium"
              >
                {link.label}
              </a>
            ))}

            {/* Settings Dropdown */}
            <div class="relative" ref={settingsRef}>
              <button
                type="button"
                onClick={() => setIsSettingsOpen(!isSettingsOpen)}
                class="p-2 hover:bg-primary-700 rounded-lg transition-colors"
                aria-label="Settings menu"
              >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
              </button>

              {isSettingsOpen && (
                <div class="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-50">
                  {settingsLinks.map((link) => (
                    <a
                      key={link.href}
                      href={link.href}
                      class="block px-4 py-2 text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                      onClick={() => setIsSettingsOpen(false)}
                    >
                      {link.label}
                    </a>
                  ))}
                </div>
              )}
            </div>
          </nav>

          {/* Mobile Menu Button - 44px touch target */}
          <button
            class="md:hidden p-3 -mr-2 touch-manipulation"
            onClick={() => setIsMenuOpen(!isMenuOpen)}
            aria-label="Toggle menu"
          >
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              {isMenuOpen ? (
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              ) : (
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
              )}
            </svg>
          </button>
        </div>

        {/* Mobile Navigation - larger touch targets */}
        {isMenuOpen && (
          <nav class="md:hidden pb-4">
            {navLinks.map((link) => (
              <a
                key={link.href}
                href={link.href}
                class="block py-3 hover:text-primary-200 hover:bg-primary-700 -mx-4 px-4 transition-colors font-medium touch-manipulation"
                onClick={() => setIsMenuOpen(false)}
              >
                {link.label}
              </a>
            ))}
            {/* Settings section divider */}
            <div class="border-t border-primary-500 my-2 pt-2">
              <span class="text-xs uppercase tracking-wider text-primary-300">Settings</span>
            </div>
            {settingsLinks.map((link) => (
              <a
                key={link.href}
                href={link.href}
                class="block py-3 hover:text-primary-200 hover:bg-primary-700 -mx-4 px-4 transition-colors font-medium touch-manipulation"
                onClick={() => setIsMenuOpen(false)}
              >
                {link.label}
              </a>
            ))}
          </nav>
        )}
      </div>
    </header>
  );
}
