import Link from "next/link";

export default function PrivacyPolicy() {
  return (
    <main className="min-h-screen bg-white text-black">
      {/* Header */}
      <header className="w-full max-w-4xl mx-auto flex justify-between items-center p-6">
        <Link href="/" className="text-xl font-bold hover:text-gray-700 transition-colors">
          Claude Control
        </Link>
      </header>

      <div className="max-w-4xl mx-auto p-6">
        <h1 className="text-4xl font-bold mb-8">Privacy Policy</h1>
        <p className="text-gray-600 mb-8">Last updated: August 11, 2025</p>

        <div className="prose max-w-none space-y-8">
          <section>
            <h2 className="text-2xl font-semibold mb-4">1. Information We Collect</h2>
            <p className="mb-4">
              Claude Control collects information to provide and improve our services. We collect:
            </p>
            <ul className="list-disc pl-6 space-y-2">
              <li>Email address and Slack/Discord account information for authentication</li>
              <li>Repository access permissions for code interaction</li>
              <li>Chat messages and commands sent to our AI agent</li>
              <li>Usage data to improve our service quality</li>
              <li>Technical information such as IP address and browser type</li>
            </ul>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">2. How We Use Your Information</h2>
            <p className="mb-4">We use the information we collect to:</p>
            <ul className="list-disc pl-6 space-y-2">
              <li>Provide AI agent services in Slack and Discord</li>
              <li>Interact with your connected repositories and create pull requests</li>
              <li>Maintain and improve our AI capabilities</li>
              <li>Communicate with you about your account and service updates</li>
              <li>Analyze usage patterns to enhance user experience</li>
              <li>Comply with legal obligations</li>
            </ul>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">3. Information Sharing</h2>
            <p className="mb-4">
              We do not sell, trade, or rent your personal information to third parties. We may share your information only in the following circumstances:
            </p>
            <ul className="list-disc pl-6 space-y-2">
              <li>With your explicit consent</li>
              <li>To comply with legal requirements or protect our rights</li>
              <li>With service providers who assist in delivering our services (under strict confidentiality agreements)</li>
              <li>In connection with a business transfer or merger</li>
            </ul>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">4. Data Security</h2>
            <p className="mb-4">
              We implement appropriate technical and organizational security measures to protect your personal information against unauthorized access, alteration, disclosure, or destruction. However, no method of transmission over the internet is 100% secure.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">5. Data Retention</h2>
            <p className="mb-4">
              We retain your personal information for as long as your account is active or as needed to provide you services. You may request deletion of your account and associated data at any time.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">6. Your Rights</h2>
            <p className="mb-4">You have the right to:</p>
            <ul className="list-disc pl-6 space-y-2">
              <li>Access your personal information</li>
              <li>Correct inaccurate information</li>
              <li>Request deletion of your data</li>
              <li>Object to processing of your data</li>
              <li>Withdraw consent at any time</li>
              <li>Export your data in a portable format</li>
            </ul>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">7. Cookies and Tracking</h2>
            <p className="mb-4">
              We use essential cookies to provide our services. We do not use tracking cookies or third-party analytics tools without your consent. You can control cookie preferences through your browser settings.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">8. Third-Party Content</h2>
            <p className="mb-4">
              Our service integrates with third-party platforms including Slack, Discord, GitHub, and various MCP servers. We are not responsible for the privacy practices of these third-party platforms. We encourage you to review their privacy policies.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">9. Children's Privacy</h2>
            <p className="mb-4">
              Claude Control is not intended for users under 13 years of age. We do not knowingly collect personal information from children under 13. If we become aware that we have collected such information, we will delete it promptly.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">10. International Users</h2>
            <p className="mb-4">
              If you are accessing our service from outside the United States, please be aware that your information may be transferred to, stored, and processed in the United States where our servers are located.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">11. Changes to This Policy</h2>
            <p className="mb-4">
              We may update this privacy policy from time to time. We will notify you of any material changes by posting the new policy on this page and updating the "Last updated" date.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">12. Contact Us</h2>
            <p className="mb-4">
              If you have any questions about this Privacy Policy or our data practices, please contact us at:
            </p>
            <p className="mb-2">Email: support@pmihaylov.com</p>
          </section>
        </div>

        <div className="mt-12 pt-8 border-t border-gray-300">
          <Link href="/" className="text-blue-600 hover:text-blue-800 transition-colors">
            ← Back to Home
          </Link>
        </div>
      </div>
    </main>
  );
}