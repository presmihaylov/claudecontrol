import { auth } from '@clerk/nextjs/server';
import { NextRequest, NextResponse } from 'next/server';
import { env } from '@/lib/env';
import * as crypto from 'node:crypto';

export async function GET(request: NextRequest) {
  console.log('Chat auth API route called');
  try {
    // Get the authenticated user from Clerk
    const { getToken } = await auth();
    const token = await getToken();
    console.log('Token obtained:', !!token);

    if (!token) {
      return NextResponse.json(
        { error: 'Authentication required' },
        { status: 401 }
      );
    }

    // Make request to ccbackend to get user profile
    console.log('Making request to:', `${env.CCBACKEND_BASE_URL}/users/profile`);
    const response = await fetch(`${env.CCBACKEND_BASE_URL}/users/profile`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
    });
    console.log('Backend response status:', response.status);

    if (!response.ok) {
      console.error('Failed to fetch user profile from backend:', response.status, response.statusText);
      return NextResponse.json(
        { error: 'Failed to fetch user profile' },
        { status: response.status }
      );
    }

    const userProfile = await response.json();

    if (!userProfile.email) {
      return NextResponse.json(
        { error: 'User email not found' },
        { status: 400 }
      );
    }

    // Generate email hash for Plain chat authentication
    const plainChatSecret = process.env.PLAIN_CHAT_SECRET;
    if (!plainChatSecret) {
      console.error('PLAIN_CHAT_SECRET environment variable is not set');
      return NextResponse.json(
        { error: 'Chat service configuration error' },
        { status: 500 }
      );
    }

    const hmac = crypto.createHmac('sha256', plainChatSecret);
    hmac.update(userProfile.email);
    const emailHash = hmac.digest('hex');

    return NextResponse.json({
      email: userProfile.email,
      emailHash: emailHash,
      fullName: userProfile.email.split('@')[0], // Use email prefix as name if no full name available
      shortName: userProfile.email.split('@')[0].split('.')[0] // Use first part of email as short name
    });
  } catch (error) {
    console.error('Error generating chat authentication:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}