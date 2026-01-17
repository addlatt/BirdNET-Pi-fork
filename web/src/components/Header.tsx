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
    { href: '/', label: 'Overview' },
    { href: '/live', label: 'Live' },
    { href: '/detections', label: 'Detections' },
    { href: '/recordings', label: 'Play' },
    { href: '/stats', label: 'Stats' },
    { href: '/species', label: 'Species' },
  ];

  const settingsLinks: NavLink[] = [
    { href: '/settings', label: 'Settings' },
    { href: '/advanced-settings', label: 'Advanced' },
    { href: '/services', label: 'Services' },
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
          <a href="/" class="flex items-center space-x-2">
            <svg class="w-8 h-8" fill="currentColor" viewBox="0 0 341.874 341.874">
              <path d="M341.338,56.544c-0.865-1.748-2.654-2.834-4.669-2.834c-0.243,0-0.471,0.016-0.662,0.035l-13.201,0.573l10.858-10.302l0.248-0.132l2.895-3.426l-1.536-3.02c-0.887-1.743-2.67-2.826-4.654-2.826c-0.489,0-0.928,0.066-1.289,0.15l-35.572,6.485c-0.648-0.844-1.308-1.672-1.984-2.479c-12.529-14.959-29.409-22.865-48.814-22.865c-4.852,0-9.917,0.497-15.056,1.476c-36.111,6.882-53.736,34.2-72.396,63.122c-24.924,38.632-50.697,78.579-124.683,78.579c-6.337,0-13.028-0.301-19.886-0.896c-0.695-0.061-1.358-0.091-1.97-0.091c-3.851,0-6.544,1.251-8.006,3.717c-2.208,3.727-0.062,7.647,0.969,9.531l0.142,0.261c13.907,25.674,34.957,47.705,60.9,63.712c23.211,14.321,49.99,23.099,76.99,25.806v52.677c0,6.747,5.517,12.171,12.265,12.171h28.832c0.799,0,1.593-0.092,2.373-0.243c0.783,0.158,1.593,0.243,2.422,0.243h28.832c3.66,0,7.256-1.609,9.62-4.359c2.116-2.461,3.024-5.496,2.556-8.58c-1.294-8.509-12.61-11.532-22.768-11.532c-2.314,0-6.642,0.184-11.032,1.307l3.213-44.469c3.401-0.65,6.804-1.365,10.205-2.186c46.987-11.342,72.971-42.049,86.494-65.814c16.654-29.266,23.972-64.827,20.076-97.568c-0.326-2.739-0.727-5.427-1.202-8.063l27.382-21.343c1.025-0.608,1.824-1.513,2.262-2.59C342.051,59.4,341.991,57.852,341.338,56.544z M173.964,301.607c-1-0.067-2.282-0.101-3.326-0.101c-2.314,0-6.727,0.18-11.117,1.303l3.079-40.844c3.728-0.08,7.365-0.271,11.365-0.568V301.607z M137.724,201.387c-15.404,10.814-31.967,11.775-41.318-4.436c-10.372,3.407-21.528,2.202-26.284-7.327c-1.491-2.988,0.775-5.469,2.189-5.541c52.375-2.654,99.886-43.521,118.922-86.605c1.398-3.165,5.691-3.562,6.524-0.52C212.89,152.204,194.946,219.858,137.724,201.387z M242.354,87.651c-9.213,0-16.682-7.469-16.682-16.682s7.469-16.682,16.682-16.682c9.213,0,16.682,7.469,16.682,16.682S251.567,87.651,242.354,87.651z"/>
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
