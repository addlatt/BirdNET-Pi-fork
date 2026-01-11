import { useState } from 'preact/hooks';
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

  const navLinks: NavLink[] = [
    { href: '/app/', label: 'Overview' },
    { href: '/app/detections', label: 'Today' },
    { href: '/app/history', label: 'History' },
    { href: '/app/stats', label: 'Stats' },
  ];

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
          <nav class="hidden md:flex space-x-8">
            {navLinks.map((link) => (
              <a
                key={link.href}
                href={link.href}
                class="hover:text-primary-200 transition-colors font-medium"
              >
                {link.label}
              </a>
            ))}
          </nav>

          {/* Mobile Menu Button */}
          <button
            class="md:hidden p-2"
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

        {/* Mobile Navigation */}
        {isMenuOpen && (
          <nav class="md:hidden pb-4">
            {navLinks.map((link) => (
              <a
                key={link.href}
                href={link.href}
                class="block py-2 hover:text-primary-200 transition-colors font-medium"
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
