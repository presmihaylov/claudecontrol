"use client";

import { useEffect } from "react";

declare global {
  interface Window {
    Plain?: {
      init: (config: any) => void;
    };
  }
}

export default function PlainChat() {
  useEffect(() => {
    // Load Plain chat script
    const script = document.createElement('script');
    script.async = false;
    script.onload = function() {
      if (window.Plain) {
        window.Plain.init({
          appId: 'liveChatApp_01K2D5ZDZ3SVFFW50SNT2FKEVY',
          
          // Enable automatic email verification
          requireAuthentication: true,
          
          // Styling with white background and black text
          theme: 'light',
          style: {
            brandColor: '#000000',
            brandBackgroundColor: '#ffffff',
            launcherBackgroundColor: '#ffffff',
            launcherIconColor: '#000000'
          },
          
          // Position the chat widget
          position: {
            right: '20px',
            bottom: '20px'
          },
          
          // Add helpful links
          links: [
            {
              icon: 'email',
              text: 'Email Support',
              url: 'mailto:support@pmihaylov.com'
            }
          ]
        });
      }
    };
    script.src = 'https://chat.cdn-plain.com/index.js';
    document.getElementsByTagName('head')[0].appendChild(script);

    // Cleanup on unmount
    return () => {
      const existingScript = document.querySelector('script[src="https://chat.cdn-plain.com/index.js"]');
      if (existingScript) {
        existingScript.remove();
      }
    };
  }, []);

  return null; // This component doesn't render anything visible
}