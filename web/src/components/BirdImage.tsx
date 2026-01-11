import type { JSX } from 'preact';
import { useState, useEffect } from 'preact/hooks';

/**
 * BirdImage props
 */
interface BirdImageProps {
  /** Scientific name of the species */
  sciName: string;
  /** Common name of the species (fallback for search) */
  comName: string;
  /** CSS class for the image container */
  class?: string;
  /** Image size (small, medium, large) */
  size?: 'small' | 'medium' | 'large';
}

/**
 * Wikipedia API response types
 */
interface WikipediaPage {
  pageid: number;
  title: string;
  thumbnail?: {
    source: string;
    width: number;
    height: number;
  };
  original?: {
    source: string;
    width: number;
    height: number;
  };
}

interface WikipediaQueryResponse {
  query?: {
    pages?: Record<string, WikipediaPage>;
  };
}

/**
 * Image cache to avoid repeated API calls
 */
const imageCache = new Map<string, string | null>();

/**
 * Size configurations
 */
const sizeConfig = {
  small: { class: 'w-10 h-10', width: 100 },
  medium: { class: 'w-16 h-16', width: 200 },
  large: { class: 'w-24 h-24', width: 300 },
};

/**
 * BirdImage - Displays a bird image fetched from Wikipedia.
 * Uses the scientific name to search for the Wikipedia article and extract the main image.
 */
export function BirdImage({ sciName, comName, class: className, size = 'medium' }: BirdImageProps): JSX.Element {
  const [imageUrl, setImageUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  const config = sizeConfig[size];

  useEffect(() => {
    let cancelled = false;

    async function fetchImage() {
      // Check cache first
      const cacheKey = sciName.toLowerCase();
      if (imageCache.has(cacheKey)) {
        setImageUrl(imageCache.get(cacheKey) ?? null);
        setLoading(false);
        return;
      }

      try {
        setLoading(true);
        setError(false);

        // Try scientific name first, then common name
        const searchTerms = [sciName, comName];
        let foundUrl: string | null = null;

        for (const term of searchTerms) {
          if (foundUrl) break;

          const url = new URL('https://en.wikipedia.org/w/api.php');
          url.searchParams.set('action', 'query');
          url.searchParams.set('titles', term);
          url.searchParams.set('prop', 'pageimages');
          url.searchParams.set('pithumbsize', String(config.width));
          url.searchParams.set('format', 'json');
          url.searchParams.set('origin', '*');
          url.searchParams.set('redirects', '1'); // Follow Wikipedia redirects

          const response = await fetch(url.toString());
          if (!response.ok) continue;

          const data = (await response.json()) as WikipediaQueryResponse;
          const pages = data.query?.pages;

          if (pages) {
            const page = Object.values(pages)[0];
            if (page?.thumbnail?.source) {
              foundUrl = page.thumbnail.source;
            }
          }
        }

        if (!cancelled) {
          imageCache.set(cacheKey, foundUrl);
          setImageUrl(foundUrl);
          setLoading(false);
        }
      } catch (err) {
        if (!cancelled) {
          imageCache.set(cacheKey, null);
          setError(true);
          setLoading(false);
        }
      }
    }

    fetchImage();

    return () => {
      cancelled = true;
    };
  }, [sciName, comName, config.width]);

  // Container classes
  const containerClasses = `${config.class} rounded-full overflow-hidden flex-shrink-0 ${className || ''}`;

  if (loading) {
    return (
      <div class={`${containerClasses} bg-gray-200 dark:bg-gray-700 animate-pulse`} />
    );
  }

  if (error || !imageUrl) {
    return (
      <div class={`${containerClasses} bg-gray-200 dark:bg-gray-700 flex items-center justify-center`}>
        <BirdPlaceholderIcon class="w-1/2 h-1/2 text-gray-400 dark:text-gray-500" />
      </div>
    );
  }

  return (
    <img
      src={imageUrl}
      alt={comName}
      class={`${containerClasses} object-cover bg-gray-200 dark:bg-gray-700`}
      loading="lazy"
      onError={() => setError(true)}
    />
  );
}

/**
 * Placeholder icon for when no image is available
 */
function BirdPlaceholderIcon({ class: className }: { class?: string }): JSX.Element {
  return (
    <svg class={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M21.9 8.89l-1.05-4.37c-.22-.9-1-1.52-1.91-1.52H5.05c-.9 0-1.69.63-1.9 1.52L2.1 8.89c-.24 1.02-.02 2.06.62 2.88.08.11.19.19.28.29V19c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2v-6.94c.09-.09.2-.18.28-.28.64-.82.87-1.87.62-2.89zm-2.99-3.9l1.05 4.37c.1.42.01.84-.25 1.17-.14.18-.44.47-.94.47-.61 0-1.14-.49-1.21-1.14L16.98 5l1.93-.01zM13 5h1.96l.54 4.52c.05.39-.07.78-.33 1.07-.22.26-.54.41-.95.41-.67 0-1.22-.59-1.22-1.31V5zM8.49 9.52L9.04 5H11v4.69c0 .72-.55 1.31-1.29 1.31-.34 0-.65-.15-.89-.41-.25-.29-.37-.68-.33-1.07zm-4.45-.16L5.05 5h1.97l-.58 4.86c-.08.65-.6 1.14-1.21 1.14-.49 0-.8-.29-.93-.47-.27-.32-.36-.75-.26-1.17zM5 19v-6.03c.08.01.15.03.23.03.87 0 1.66-.36 2.24-.95.6.6 1.4.95 2.31.95.87 0 1.65-.36 2.23-.93.59.57 1.39.93 2.29.93.84 0 1.64-.35 2.24-.95.58.59 1.37.95 2.24.95.08 0 .15-.02.23-.03V19H5z" />
    </svg>
  );
}
