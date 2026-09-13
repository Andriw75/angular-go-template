import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { SidebarComponent } from '../sidebar/sidebar';
import { ConfirmComponent } from '../../../common/confirm/confirm';
import { ToastComponent } from '../../../common/toast/toast';

@Component({
  selector: 'app-dashboard-layout',
  templateUrl: './layout.html',
  styleUrl: './layout.css',
  imports: [RouterOutlet, SidebarComponent, ConfirmComponent, ToastComponent],
})
export class DashboardLayoutComponent {}
