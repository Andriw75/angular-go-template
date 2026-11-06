import { Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { environment } from '../../../../../environments/environment';

@Component({
  selector: 'app-brand-logo',
  templateUrl: './brand-logo.html',
  styleUrl: './brand-logo.css',
  imports: [RouterLink],
})
export class BrandLogoComponent {
  readonly name = environment.APP_NAME;
}
